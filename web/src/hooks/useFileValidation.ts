/**
 * useFileValidation Hook
 * Validation for file uploads with configurable rules
 * 
 * Validates:
 * - File size (max bytes)
 * - File type (MIME type whitelist)
 * - File extension (extension whitelist)
 * - Filename sanitization
 */

import { useCallback } from 'react';

/**
 * File validation options
 */
export interface FileValidationOptions {
  /** Maximum file size in bytes */
  maxSize?: number;
  /** Allowed MIME types */
  allowedMimeTypes?: string[];
  /** Allowed file extensions */
  allowedExtensions?: string[];
  /** Custom validation function */
  customValidator?: (file: File) => string | null;
}

/**
 * File validation result
 */
export interface FileValidationResult {
  /** Whether the file is valid */
  isValid: boolean;
  /** Error message if invalid */
  error: string | null;
  /** Sanitized filename */
  sanitizedName: string;
}

/**
 * Default validation options (conservative)
 */
const DEFAULT_OPTIONS: FileValidationOptions = {
  maxSize: 10 * 1024 * 1024, // 10MB
  allowedMimeTypes: [
    'image/jpeg',
    'image/png',
    'image/webp',
    'image/gif',
    'audio/mpeg',
    'audio/wav',
    'audio/ogg',
    'video/mp4',
    'video/webm',
  ],
  allowedExtensions: [
    'jpg', 'jpeg', 'png', 'webp', 'gif',
    'mp3', 'wav', 'ogg',
    'mp4', 'webm',
  ],
};

/**
 * Sanitize filename to prevent directory traversal and script injection
 * Removes special characters and normalizes the name
 */
export function sanitizeFilename(filename: string): string {
  // Remove path separators and null bytes
  let sanitized = filename
    .replace(/[\/\\:*?"<>|]/g, '_')
    .replace(/\0/g, '')
    .trim();

  // Remove leading dots (prevents hidden files)
  sanitized = sanitized.replace(/^\.+/, '');

  // Keep only safe characters: alphanumeric, dash, underscore, dot
  sanitized = sanitized.replace(/[^a-zA-Z0-9._\- ]/g, '_');

  // Collapse multiple spaces and underscores
  sanitized = sanitized.replace(/[\s_]+/g, '_');

  // Remove trailing dots/spaces
  sanitized = sanitized.replace(/[\s.]+$/, '');

  // If empty after sanitization, use default
  return sanitized || 'file';
}

/**
 * Extract file extension safely
 */
function getFileExtension(filename: string): string {
  const parts = filename.split('.');
  if (parts.length < 2) return '';
  return parts[parts.length - 1].toLowerCase();
}

/**
 * Hook for file upload validation
 * 
 * @param options - Validation options
 * @returns Validation function
 * 
 * @example
 * ```tsx
 * const validate = useFileValidation({
 *   maxSize: 5 * 1024 * 1024,
 *   allowedMimeTypes: ['image/jpeg', 'image/png'],
 * });
 * 
 * const handleFileSelect = (file: File) => {
 *   const result = validate(file);
 *   if (!result.isValid) {
 *     console.error(result.error);
 *   }
 * };
 * ```
 */
export function useFileValidation(options: FileValidationOptions = {}): (file: File) => FileValidationResult {
  const mergedOptions = { ...DEFAULT_OPTIONS, ...options };

  const validate = useCallback(
    (file: File): FileValidationResult => {
      const sanitizedName = sanitizeFilename(file.name);
      const fileExt = getFileExtension(file.name).toLowerCase();

      // Check file size
      if (mergedOptions.maxSize && file.size > mergedOptions.maxSize) {
        const maxSizeMB = mergedOptions.maxSize / (1024 * 1024);
        return {
          isValid: false,
          error: `File size exceeds ${maxSizeMB}MB limit (file is ${(file.size / (1024 * 1024)).toFixed(2)}MB)`,
          sanitizedName,
        };
      }

      // Check MIME type
      if (
        mergedOptions.allowedMimeTypes &&
        mergedOptions.allowedMimeTypes.length > 0 &&
        !mergedOptions.allowedMimeTypes.includes(file.type)
      ) {
        return {
          isValid: false,
          error: `File type "${file.type}" is not allowed. Allowed types: ${mergedOptions.allowedMimeTypes.join(', ')}`,
          sanitizedName,
        };
      }

      // Check file extension
      if (
        mergedOptions.allowedExtensions &&
        mergedOptions.allowedExtensions.length > 0 &&
        !mergedOptions.allowedExtensions.includes(fileExt)
      ) {
        return {
          isValid: false,
          error: `File extension ".${fileExt}" is not allowed. Allowed extensions: ${mergedOptions.allowedExtensions.map(e => `.${e}`).join(', ')}`,
          sanitizedName,
        };
      }

      // Run custom validator if provided
      if (mergedOptions.customValidator) {
        const customError = mergedOptions.customValidator(file);
        if (customError) {
          return {
            isValid: false,
            error: customError,
            sanitizedName,
          };
        }
      }

      return {
        isValid: true,
        error: null,
        sanitizedName,
      };
    },
    [mergedOptions]
  );

  return validate;
}

/**
 * Validation presets for common use cases
 */
export const FileValidationPresets = {
  /** Image files only */
  images: {
    maxSize: 10 * 1024 * 1024, // 10MB
    allowedMimeTypes: ['image/jpeg', 'image/png', 'image/webp', 'image/gif'],
    allowedExtensions: ['jpg', 'jpeg', 'png', 'webp', 'gif'],
  },

  /** Audio files only */
  audio: {
    maxSize: 100 * 1024 * 1024, // 100MB
    allowedMimeTypes: ['audio/mpeg', 'audio/wav', 'audio/ogg', 'audio/flac'],
    allowedExtensions: ['mp3', 'wav', 'ogg', 'flac'],
  },

  /** Video files only */
  video: {
    maxSize: 500 * 1024 * 1024, // 500MB
    allowedMimeTypes: ['video/mp4', 'video/webm', 'video/mpeg'],
    allowedExtensions: ['mp4', 'webm', 'mpg', 'mpeg'],
  },

  /** Documents only */
  documents: {
    maxSize: 50 * 1024 * 1024, // 50MB
    allowedMimeTypes: [
      'application/pdf',
      'application/msword',
      'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      'text/plain',
    ],
    allowedExtensions: ['pdf', 'doc', 'docx', 'txt'],
  },
};
