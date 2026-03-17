# Legal & Compliance Documentation

This directory contains privacy, legal, and compliance documentation for the Subcults platform.

## 📋 Document Index

### Core Legal Documents

1. **[Privacy Policy](./PRIVACY_POLICY.md)**
   - Data collection practices
   - Third-party services and data sharing
   - User rights and controls
   - Security measures
   - International compliance (GDPR, CCPA)
   - Contact information for privacy inquiries

2. **[Terms of Service](./TERMS_OF_SERVICE.md)**
   - Acceptable use policy
   - User accounts and responsibilities
   - Content ownership and licensing
   - Payment terms (Stripe Connect)
   - Disclaimers and liability limitations
   - Dispute resolution procedures

### Compliance Guides

3. **[GDPR Compliance Guide](./GDPR_COMPLIANCE_GUIDE.md)**
   - EU data protection rights (access, rectification, erasure, portability)
   - Data subject request procedures
   - Lawful basis for processing
   - International data transfers
   - Supervisory authority contact information
   - Data breach notification protocols

4. **[Data Retention Policy](./DATA_RETENTION_POLICY.md)**
   - Retention periods by data category
   - Archival procedures for cold storage
   - Deletion procedures (soft delete and hard delete)
   - Automated cleanup jobs
   - Backup retention and disaster recovery
   - Third-party processor retention policies

## 🎯 Quick Reference

### For Users

- **Privacy Questions:** [Privacy Policy](./PRIVACY_POLICY.md) → Section 11 (Contact)
- **Data Access Request:** [GDPR Guide](./GDPR_COMPLIANCE_GUIDE.md) → Section 3.1
- **Account Deletion:** [GDPR Guide](./GDPR_COMPLIANCE_GUIDE.md) → Section 3.3
- **Report Privacy Issue:** privacy@subcults.org
- **Report Security Issue:** security@subcults.org

### For Developers

- **Technical Privacy Implementation:** [../PRIVACY.md](../PRIVACY.md)
- **Location Consent Enforcement:** Search for `EnforceLocationConsent()` in codebase
- **EXIF Stripping:** `internal/image/` package
- **Data Retention Jobs:** [Data Retention Policy](./DATA_RETENTION_POLICY.md) → Section 4

### For Legal/Compliance Teams

- **GDPR Response SLA:** 30 days ([GDPR Guide](./GDPR_COMPLIANCE_GUIDE.md) → Section 4.1)
- **Data Breach Notification:** 72 hours to authority, immediate to users ([GDPR Guide](./GDPR_COMPLIANCE_GUIDE.md) → Section 7)
- **Retention Periods Summary:** [Data Retention Policy](./DATA_RETENTION_POLICY.md) → Section 1
- **Third-Party DPAs:** Contact privacy@subcults.org

## 🔒 Key Privacy Principles

1. **Privacy First**
   - Location precision requires explicit opt-in (`allow_precise` flag)
   - Default: Coarse geohash (~±0.61 km) with deterministic jitter
   - EXIF metadata stripped from all media uploads

2. **User Control**
   - Granular consent for location, telemetry, session replay
   - One-click opt-out for analytics
   - Data export in machine-readable format (JSON)

3. **Data Minimization**
   - No IP logging (except transiently for rate limiting)
   - No request body logging
   - No browsing history tracking
   - PII redacted from client error logs

4. **Transparency**
   - Clear retention periods for all data categories
   - Third-party services documented with privacy policy links
   - Compliance reporting and transparency reports

## 📞 Contact Information

| Purpose | Email | Response Time |
|---------|-------|---------------|
| General Privacy Inquiries | privacy@subcults.org | 7 business days |
| Data Subject Requests (GDPR) | privacy@subcults.org | 30 days |
| Data Protection Officer | dpo@subcults.org | 30 days |
| Security Issues | security@subcults.org | 24 hours |
| DMCA Copyright Claims | dmca@subcults.org | 72 hours |
| Legal Matters | legal@subcults.org | 30 days |

**Mailing Address:**  
Subcults Legal Department  
548 Market Street, PMB 12345  
San Francisco, CA 94104  
United States

## ⚖️ Regulatory Compliance

### GDPR (EU/EEA/UK)
- **Lawful Basis:** Contract, consent, legitimate interest, legal obligation
- **Data Protection Officer:** dpo@subcults.org
- **Supervisory Authority:** [Find Your Authority](https://edpb.europa.eu/about-edpb/board/members_en)
- **Response SLA:** 30 days for data subject requests

### CCPA (California, USA)
- **Right to Know:** Request categories of data collected
- **Right to Delete:** Request deletion of personal information
- **Right to Opt-Out:** We do NOT sell personal data
- **Contact:** privacy@subcults.org (Subject: "CCPA Request")

### Other Jurisdictions
We comply with data protection laws in all jurisdictions where we operate. Contact privacy@subcults.org for jurisdiction-specific inquiries.

## 🔄 Document Updates

All legal documents are reviewed and updated:
- **Annually:** As part of compliance review cycle
- **Ad Hoc:** When regulations change or new features are introduced
- **Notification:** Material changes communicated 30 days in advance via email

**Version History:**
- **v1.0 (Feb 1, 2026):** Initial legal documentation suite

## 📚 Related Documentation

- **[Technical Privacy Overview](../PRIVACY.md)** - Implementation details for developers
- **[Architecture](../ARCHITECTURE.md)** - System architecture and third-party integrations
- **[Configuration](../CONFIGURATION.md)** - Third-party service setup and credentials
- **[Main README](../../README.md)** - Project overview and getting started

## ⚠️ Legal Disclaimer

**IMPORTANT:** These documents are provided for transparency and user education. They should be reviewed and approved by qualified legal counsel before production deployment.

**Geographic Scope:** These documents are designed for global use but may require modification for specific jurisdictions. Consult local counsel for:
- EU/EEA member states with stricter age requirements (16 vs 13)
- Countries with sector-specific regulations (e.g., music licensing, payment processing)
- Jurisdictions with data localization requirements

**Professional Review Required:**
- Privacy attorney for Privacy Policy and GDPR Compliance Guide
- Contract attorney for Terms of Service
- Data protection specialist for Data Retention Policy
- Compliance officer for regulatory alignment

---

**Last Updated:** February 1, 2026  
**Maintained By:** Subcults Legal & Privacy Team  
**Questions:** legal@subcults.org
