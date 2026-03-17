package indexer

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/fxamacker/cbor/v2"
)

// CAR file format errors.
var (
	ErrInvalidCARHeader  = errors.New("invalid CAR header")
	ErrUnsupportedCARVer = errors.New("unsupported CAR version")
	ErrInvalidBlock      = errors.New("invalid CAR block")
	ErrCARTruncated      = errors.New("CAR file truncated")
)

// CARHeader represents the header of a CAR v1 file.
type CARHeader struct {
	Version int      `cbor:"version"`
	Roots   [][]byte `cbor:"roots"`
}

// CARBlock represents a single block from a CAR file (CID + data).
type CARBlock struct {
	Offset int64
	CID    []byte
	Data   []byte
}

// CARReader provides streaming access to blocks in a CAR v1 file.
type CARReader struct {
	reader io.Reader
	header *CARHeader
	offset int64
	logger *slog.Logger
}

// NewCARReader creates a reader that parses CAR v1 format.
// It reads and validates the header immediately.
func NewCARReader(r io.Reader, logger *slog.Logger) (*CARReader, error) {
	if logger == nil {
		logger = slog.Default()
	}
	cr := &CARReader{reader: r, logger: logger}
	if err := cr.readHeader(); err != nil {
		return nil, err
	}
	return cr, nil
}

// Header returns the parsed CAR header.
func (cr *CARReader) Header() *CARHeader {
	return cr.header
}

// Next reads the next block from the CAR file.
// Returns io.EOF when no more blocks are available.
func (cr *CARReader) Next() (*CARBlock, error) {
	// Read varint-encoded section length
	length, err := binary.ReadUvarint(cr)
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("%w: failed to read block length: %v", ErrCARTruncated, err)
	}
	if length == 0 {
		return nil, io.EOF
	}
	if length > 4*1024*1024 { // 4MB max block size
		return nil, fmt.Errorf("%w: block too large: %d bytes", ErrInvalidBlock, length)
	}

	blockStart := cr.offset
	section := make([]byte, length)
	if _, err := io.ReadFull(cr.reader, section); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCARTruncated, err)
	}
	cr.offset += int64(length)

	// Parse CID from section start
	cidLen, cidData, err := parseCID(section)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBlock, err)
	}

	return &CARBlock{
		Offset: blockStart,
		CID:    cidData[:cidLen],
		Data:   section[cidLen:],
	}, nil
}

// ReadByte implements io.ByteReader for binary.ReadUvarint.
func (cr *CARReader) ReadByte() (byte, error) {
	buf := make([]byte, 1)
	_, err := io.ReadFull(cr.reader, buf)
	if err != nil {
		return 0, err
	}
	cr.offset++
	return buf[0], nil
}

func (cr *CARReader) readHeader() error {
	// Read varint-encoded header length
	headerLen, err := binary.ReadUvarint(cr)
	if err != nil {
		return fmt.Errorf("%w: failed to read header length: %v", ErrInvalidCARHeader, err)
	}
	if headerLen == 0 || headerLen > 1024*1024 { // 1MB max header
		return fmt.Errorf("%w: invalid header length: %d", ErrInvalidCARHeader, headerLen)
	}

	headerBytes := make([]byte, headerLen)
	if _, err := io.ReadFull(cr.reader, headerBytes); err != nil {
		return fmt.Errorf("%w: truncated header: %v", ErrInvalidCARHeader, err)
	}
	cr.offset += int64(headerLen)

	var header CARHeader
	if err := cbor.Unmarshal(headerBytes, &header); err != nil {
		return fmt.Errorf("%w: invalid CBOR header: %v", ErrInvalidCARHeader, err)
	}
	if header.Version != 1 {
		return fmt.Errorf("%w: version %d", ErrUnsupportedCARVer, header.Version)
	}
	cr.header = &header
	cr.logger.Debug("parsed CAR header", "version", header.Version, "roots", len(header.Roots))
	return nil
}

// parseCID extracts a CID from the beginning of a byte slice.
// Returns the length consumed and the CID bytes.
func parseCID(data []byte) (int, []byte, error) {
	if len(data) < 2 {
		return 0, nil, fmt.Errorf("section too short for CID")
	}
	// CIDv1: multibase prefix + version varint + codec varint + multihash
	// Minimal: version(1) + codec(1) + hash-fn(1) + hash-len(1) + hash(N)
	pos := 0

	// Read CID version varint
	version, n := binary.Uvarint(data[pos:])
	if n <= 0 {
		return 0, nil, fmt.Errorf("invalid CID version varint")
	}
	pos += n

	if version == 0x12 {
		// CIDv0: raw sha256 multihash (starts with 0x12 0x20)
		// 0x12 = sha2-256, next byte = digest length
		if pos >= len(data) {
			return 0, nil, fmt.Errorf("CIDv0 truncated")
		}
		digestLen := int(data[pos])
		pos++
		cidLen := pos + digestLen
		if cidLen > len(data) {
			return 0, nil, fmt.Errorf("CIDv0 digest truncated")
		}
		return cidLen, data[:cidLen], nil
	}

	// CIDv1: version + codec + multihash
	// Read codec varint
	_, n = binary.Uvarint(data[pos:])
	if n <= 0 {
		return 0, nil, fmt.Errorf("invalid CID codec varint")
	}
	pos += n

	// Read multihash: function varint + length varint + digest
	_, n = binary.Uvarint(data[pos:])
	if n <= 0 {
		return 0, nil, fmt.Errorf("invalid multihash function varint")
	}
	pos += n

	digestLen, n := binary.Uvarint(data[pos:])
	if n <= 0 {
		return 0, nil, fmt.Errorf("invalid multihash digest length varint")
	}
	pos += n

	cidLen := pos + int(digestLen)
	if cidLen > len(data) {
		return 0, nil, fmt.Errorf("multihash digest truncated")
	}
	return cidLen, data[:cidLen], nil
}

// CARImporter streams blocks from a CAR file and processes AT Protocol records.
type CARImporter struct {
	filter  *RecordFilter
	repo    RecordRepository
	logger  *slog.Logger
}

// NewCARImporter creates a CAR file importer.
func NewCARImporter(repo RecordRepository, filter *RecordFilter, logger *slog.Logger) *CARImporter {
	if logger == nil {
		logger = slog.Default()
	}
	return &CARImporter{
		filter: filter,
		repo:   repo,
		logger: logger,
	}
}

// CARImportResult holds the outcome of a CAR import.
type CARImportResult struct {
	BlocksRead       int64
	RecordsProcessed int64
	RecordsSkipped   int64
	Errors           int64
}

// Import reads all blocks from a CAR reader and processes matching records.
func (ci *CARImporter) Import(ctx context.Context, reader *CARReader, dryRun bool) (*CARImportResult, error) {
	result := &CARImportResult{}

	for {
		select {
		case <-ctx.Done():
			ci.logger.Info("CAR import cancelled", "blocks_read", result.BlocksRead)
			return result, ctx.Err()
		default:
		}

		block, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors++
			ci.logger.Warn("error reading CAR block", "error", err, "offset", reader.offset)
			continue
		}

		result.BlocksRead++

		// Try to decode block as AT Protocol commit
		if err := ci.processBlock(ctx, block, dryRun, result); err != nil {
			result.Errors++
			ci.logger.Debug("skipping non-record block", "offset", block.Offset, "error", err)
		}
	}

	ci.logger.Info("CAR import complete",
		"blocks_read", result.BlocksRead,
		"records_processed", result.RecordsProcessed,
		"records_skipped", result.RecordsSkipped,
		"errors", result.Errors,
	)
	return result, nil
}

func (ci *CARImporter) processBlock(ctx context.Context, block *CARBlock, dryRun bool, result *CARImportResult) error {
	// Try to decode as an AT Protocol record embedded in CBOR
	var record map[string]interface{}
	if err := cbor.Unmarshal(block.Data, &record); err != nil {
		return fmt.Errorf("not a CBOR record: %w", err)
	}

	// Check for $type field (AT Protocol record marker)
	typeVal, ok := record["$type"]
	if !ok {
		return fmt.Errorf("no $type field")
	}
	collection, ok := typeVal.(string)
	if !ok {
		return fmt.Errorf("$type is not a string")
	}

	// Filter through the record filter
	filterResult := ci.filter.Filter(collection, block.Data)
	if !filterResult.Matched {
		result.RecordsSkipped++
		return nil
	}
	if !filterResult.Valid {
		result.Errors++
		return fmt.Errorf("validation failed: %w", filterResult.Error)
	}

	filterResult.Operation = "create"

	if dryRun {
		ci.logger.Debug("dry-run: would import record", "collection", collection)
		result.RecordsProcessed++
		return nil
	}

	if _, _, err := ci.repo.UpsertRecord(ctx, &filterResult); err != nil {
		return fmt.Errorf("upsert failed: %w", err)
	}
	result.RecordsProcessed++
	return nil
}
