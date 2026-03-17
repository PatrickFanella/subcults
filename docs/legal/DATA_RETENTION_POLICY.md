# Data Retention Policy

**Effective Date:** February 1, 2026  
**Last Updated:** February 1, 2026

## Introduction

This Data Retention Policy describes how Subcults retains, archives, and deletes personal data and user content. This policy ensures compliance with GDPR, CCPA, and other data protection regulations while balancing operational, legal, and security requirements.

**Purpose:** To establish clear retention periods for all data categories, ensuring we:
1. Retain data only as long as necessary for legitimate purposes
2. Comply with legal and regulatory requirements
3. Protect user privacy and minimize data exposure
4. Enable timely deletion and user data rights

## 1. Data Categories & Retention Periods

### 1.1 User Account Data

| Data Type | Retention Period | Rationale |
|-----------|------------------|-----------|
| **DID & Profile** | Until account deletion | Core identity and platform access |
| **Authentication Tokens** | Access: 15 minutes<br>Refresh: 7 days | Security best practice for token expiry |
| **Password Hashes** | N/A | We use AT Protocol DIDs, not passwords |
| **Email Addresses** | Until account deletion | Communication and account recovery |
| **Account Creation Date** | Until account deletion + 30 days | Audit trail for account lifecycle |
| **Last Login Date** | Until account deletion + 30 days | Security monitoring and inactive account detection |

**Deletion Trigger:**
- User-initiated deletion request
- Inactive account (no login for 2+ years) → warning email → deletion after 90 days if no response

**Soft Delete Grace Period:** 30 days (allows accidental recovery)

**Permanent Deletion:** Account data is hard deleted 30 days after soft delete, except:
- Anonymized usage statistics (aggregated, no PII)
- Transaction metadata (retained 7 years for tax compliance)

### 1.2 Content Data

| Data Type | Retention Period | Rationale |
|-----------|------------------|-----------|
| **Scenes** | Until user deletion or scene removal | User-created content |
| **Events** | Until user deletion or event removal | User-created content |
| **Posts** | Until user deletion or post removal | User-created content |
| **Comments** | Until user deletion or parent content removal | User-created content |
| **Media Uploads** | Until user deletion or content removal | Supporting content |
| **Deleted Content (Soft)** | 30 days in soft-delete state | Grace period for accidental deletion |
| **Media CDN Cache** | Up to 90 days | Third-party CDN retention (Cloudflare R2) |

**Deletion Triggers:**
- User deletes individual content item
- User deletes account (cascades to all content)
- Content moderation removal (immediate hard delete for policy violations)

**Orphaned Content:** If a scene owner deletes their account, scenes may be:
- Transferred to a co-organizer (if designated)
- Archived and made read-only
- Deleted after 90 days if no transfer or co-organizer

### 1.3 Location Data

| Data Type | Retention Period | Rationale |
|-----------|------------------|-----------|
| **Coarse Geohash** | Until scene/event deletion | Regional discovery without precise location |
| **Precise Coordinates** | Until scene/event deletion OR consent withdrawal | Opt-in only; cleared immediately when `allow_precise = FALSE` |
| **Location Change History** | Not retained | We do not track location changes over time |

**Consent Enforcement:**
- Precise coordinates are **NEVER** stored without `allow_precise = TRUE`
- Database constraint: `CHECK (allow_precise = TRUE OR precise_point IS NULL)`
- Application enforcement: `EnforceLocationConsent()` clears precise data before persistence

**Deletion Triggers:**
- User toggles `allow_precise` to FALSE (immediate clear of precise coordinates)
- User deletes scene/event
- User deletes account

### 1.4 Transaction & Payment Data

| Data Type | Retention Period | Rationale |
|-----------|------------------|-----------|
| **Transaction Metadata** | 7 years | Tax and financial regulations (IRS, EU VAT) |
| **Stripe Connect Account ID** | Until account deletion + 7 years | Financial record-keeping and dispute resolution |
| **Payment Credentials** | Never stored | Processed exclusively by Stripe |
| **Refund Records** | 7 years | Dispute resolution and financial auditing |
| **Invoice Data** | 7 years | Tax compliance and accounting |

**Note:** Full payment card details are **NEVER** stored by Subcults. Stripe processes all sensitive payment information.

**Deletion Triggers:**
- 7 years after last transaction (automatic archival)
- User account deletion → transaction metadata anonymized (amounts, dates retained for compliance; user identity removed)

**GDPR Conflict Resolution:** Financial records are retained for legal compliance (GDPR Art. 17.3.b - legal obligation) even after account deletion. User identity is anonymized where possible.

### 1.5 Security & Audit Logs

| Data Type | Retention Period | Rationale |
|-----------|------------------|-----------|
| **Access Logs** | 90 days | Security monitoring, abuse detection, incident response |
| **Error Logs (Server)** | 90 days | Debugging, stability analysis, security auditing |
| **Error Logs (Client)** | 30 days | Bug fixing, UX improvement |
| **Session Replay Data** | 7 days | UX debugging (opt-in only) |
| **Authentication Events** | 90 days | Security monitoring (login attempts, token issuance) |
| **Rate Limit Events** | 7 days | Abuse detection and rate limit tuning |
| **Moderation Actions** | 1 year | Accountability and appeal resolution |

**What Is Logged:**
- Request ID, HTTP method, path (no query params), status code, latency, response size
- User DID (if authenticated)
- Error codes and stack traces (PII redacted)

**What Is NOT Logged:**
- IP addresses (except transiently for rate limiting, discarded after 7 days)
- Request bodies or form data
- Authentication credentials or tokens
- Full URLs with sensitive query parameters

**Deletion Triggers:**
- Automatic deletion after retention period
- User account deletion → logs retained for full period (security audit trail), then DID is anonymized

**GDPR Conflict Resolution:** Access logs serve a legitimate interest in security and fraud prevention (GDPR Art. 6.1.f). Logs are minimized (no IPs, no request bodies) and retained for a limited period (90 days).

### 1.6 Telemetry & Analytics

| Data Type | Retention Period | Rationale |
|-----------|------------------|-----------|
| **Usage Events** | 90 days (individual)<br>Indefinite (aggregated) | Feature engagement, performance monitoring |
| **Performance Metrics** | 90 days (detailed)<br>1 year (aggregated) | Platform optimization and capacity planning |
| **A/B Test Data** | 180 days | Experiment analysis and feature validation |
| **Crash Reports** | 30 days | Stability improvements |

**Privacy Safeguards:**
- Individual event data is deleted after 90 days
- After 90 days, data is aggregated and anonymized (no user identifiers)
- Aggregated data is retained indefinitely for trend analysis

**Opt-In:** Users can enable telemetry via Settings → Privacy → Analytics (disabled by default)
- **Effect:** Analytics events collected only when opted-in
- **Existing Data:** Deleted immediately upon opt-out (or anonymized within 90 days)

**Session Replay:**
- **Default:** OFF (opt-in only)
- **Retention:** 7 days, then auto-deleted
- **Opt-Out:** Immediate deletion of all existing replay data

### 1.7 Alliance & Trust Graph Data

| Data Type | Retention Period | Rationale |
|-----------|------------------|-----------|
| **Alliance Relationships** | Until alliance is dissolved or account deleted | Trust-based discovery and ranking |
| **Trust Scores (Cached)** | 7 days | Performance optimization for search and feed |
| **Trust Computation Logs** | 30 days | Debugging ranking algorithm |

**Deletion Triggers:**
- User revokes alliance
- User deletes account (all alliances removed)
- Alliance becomes inactive (no mutual engagement for 1 year) → warning → deletion after 90 days

**Cascade Effects:**
- Deleting an alliance recalculates trust scores for affected users
- Trust score cache is invalidated immediately

### 1.8 Communication & Notifications

| Data Type | Retention Period | Rationale |
|-----------|------------------|-----------|
| **Email Records** | 180 days | Delivery confirmation, bounce tracking, support history |
| **Web Push Subscriptions** | Until user unsubscribes or account deleted | Push notification delivery |
| **Notification Preferences** | Until account deletion | User communication settings |
| **Support Tickets** | 2 years | Customer support history and quality assurance |

**Deletion Triggers:**
- User unsubscribes from emails or push notifications
- User deletes account
- Automatic deletion after retention period

**Support Ticket Exception:** Tickets may be retained longer if:
- Active dispute or legal claim
- Regulatory investigation
- User explicitly requests retention for ongoing issue

## 2. Archival Procedures

### 2.1 Cold Storage Migration
Data that is rarely accessed but must be retained for legal compliance is archived to cold storage:

**Eligible Data:**
- Transaction metadata older than 3 years
- Moderation actions older than 1 year
- Support tickets older than 1 year

**Archive Process:**
1. **Criteria Check:** Automated daily job identifies data eligible for archival
2. **Compression:** Data is compressed and encrypted
3. **Migration:** Moved to Cloudflare R2 cold storage tier
4. **Index Retention:** Minimal index retained in primary database for retrieval
5. **Verification:** Archive integrity verified via checksum

**Archive Format:**
- **Format:** JSON with gzip compression
- **Encryption:** AES-256 at rest (managed by Cloudflare)
- **Access:** Requires manual restoration request (3-5 hour SLA)

**Retention in Archive:**
- Transaction metadata: 7 years total (3 years hot + 4 years cold)
- Moderation actions: 1 year total (6 months hot + 6 months cold)
- Support tickets: 2 years total (1 year hot + 1 year cold)

### 2.2 Archive Access Requests
Users may request archived data via:
1. Email privacy@subcults.org with "Archive Access Request"
2. Include DID, date range, and data category
3. We will restore data within 72 hours and provide download link
4. Download link expires after 7 days

**Cost:** First archive access request per year is free; subsequent requests may incur a reasonable fee to cover restoration costs.

## 3. Deletion Procedures

### 3.1 Soft Delete Process

**Trigger:** User initiates deletion of content or account

**Step 1: Mark as Deleted**
- Content/account marked with `deleted_at` timestamp
- Content becomes inaccessible to all users (except owner can view in "Recently Deleted")
- DID authentication still works (for recovery)

**Step 2: Grace Period (30 Days)**
- User can recover content via "Recently Deleted" section
- Or re-activate account by logging in and confirming

**Step 3: Hard Delete (After 30 Days)**
- Permanent deletion of data from primary database
- Cascade to related data (media files, associations, logs)
- Email confirmation sent to user

**Exception:** Immediate hard delete for:
- Content removed for policy violations (no grace period)
- User explicitly selects "Delete Immediately" option

### 3.2 Hard Delete Process

**Trigger:** End of soft delete grace period or immediate deletion request

**Step 1: Cascading Deletion**
- Identify all related data (scenes, events, posts, media, alliances, memberships)
- Delete in dependency order (children before parents)

**Step 2: Media Deletion**
- Delete media files from Cloudflare R2
- Invalidate CDN cache entries
- Verify deletion via API response

**Step 3: Database Purge**
- Delete rows from all tables (accounts, scenes, events, posts, etc.)
- Update foreign key references (set to NULL or cascade)

**Step 4: Anonymization (Where Deletion Is Prohibited)**
- Transaction metadata: Replace DID with anonymized identifier `user_deleted_YYYYMMDD_<hash>`
- Audit logs: Replace DID with `<anonymized>`
- Aggregated analytics: Already anonymized (no action needed)

**Step 5: Third-Party Cleanup**
- Stripe: Disconnect Connect account (if applicable)
- LiveKit: Revoke room access grants
- AT Protocol: No action (DID-based identity is external to Subcults)

**Step 6: Verification**
- Automated script verifies no PII remains in database
- Manual review for accounts with >1000 scenes or >10,000 transactions

### 3.3 Right to Be Forgotten (GDPR)

For GDPR "Right to Erasure" requests:

**Procedure:**
1. User submits request via privacy@subcults.org
2. Identity verification (via AT Protocol DID authentication)
3. Immediate soft delete (no 30-day grace period for GDPR requests)
4. Hard delete within 72 hours
5. Email confirmation with deletion certificate

**Exceptions (Refusal to Delete):**
- Transaction records required by law (retained 7 years, anonymized)
- Data needed for legal claims or defense
- Public interest data (anonymized aggregates only)

**Response Time:** 30 days maximum (GDPR Art. 17)

## 4. Automated Deletion Jobs

### 4.1 Scheduled Cleanup Jobs

**Daily Jobs (3:00 AM UTC):**
- Delete access logs older than 90 days
- Delete client error logs older than 30 days
- Delete session replay data older than 7 days
- Delete rate limit events older than 7 days
- Process soft-deleted content reaching 30-day mark
- Archive transaction metadata older than 3 years

**Weekly Jobs (Sunday 2:00 AM UTC):**
- Delete telemetry events older than 90 days (retain aggregates)
- Delete trust score cache older than 7 days
- Delete trust computation logs older than 30 days
- Delete email delivery records older than 180 days

**Monthly Jobs (1st of month, 1:00 AM UTC):**
- Archive support tickets older than 1 year
- Archive moderation actions older than 1 year
- Identify inactive accounts (no login for 2 years) → send warning email
- Delete inactive accounts (no response to warning after 90 days)

**Quarterly Jobs (1st of quarter, 12:00 AM UTC):**
- Vacuum database to reclaim space from deleted records
- Audit retention compliance (verify all jobs executed successfully)
- Generate compliance report (record counts per data category)

### 4.2 Job Monitoring

**Alerts:**
- Job failure (Slack/email notification to ops team)
- Deletion count anomaly (>10% deviation from average)
- Archive restoration failure

**Audit Logging:**
- All deletion jobs logged with:
  - Job ID and timestamp
  - Data category and count of records deleted
  - Execution duration
  - Success/failure status

**Compliance Dashboard:**
- Real-time view of data retention metrics
- Counts per data category and age
- Upcoming deletions and archival events

## 5. Data Backup & Disaster Recovery

### 5.1 Backup Retention

**Incremental Backups:**
- **Frequency:** Daily
- **Retention:** 30 days
- **Purpose:** Point-in-time recovery for recent data loss

**Full Backups:**
- **Frequency:** Weekly (Sunday)
- **Retention:** 90 days
- **Purpose:** Major disaster recovery

**Archive Backups:**
- **Frequency:** Monthly
- **Retention:** 1 year
- **Purpose:** Long-term compliance and historical recovery

**Encryption:** All backups encrypted at rest (AES-256)

**Storage Location:** Geographically separated from primary database (different region)

### 5.2 Backup Deletion Conflict

**Challenge:** Backups may contain data that users have requested to be deleted.

**Resolution:**
- Backups are retained for disaster recovery (legitimate interest)
- Deleted data is marked in live database (prevents restoration)
- If backup is restored, deletion markers are re-applied
- Backups older than 1 year are archived in compliance with retention policy

**GDPR Compliance:**
- Backups are not "active processing" (GDPR recital 49)
- Reasonable backup retention (90 days) for disaster recovery is permitted
- User deletion requests apply to live database immediately

## 6. Third-Party Data Retention

### 6.1 Processor Retention Policies

We contractually require all data processors to:
- Retain data only as long as necessary for service provision
- Delete data within 30 days of contract termination
- Provide deletion confirmation upon request

**Third-Party Retention Periods:**
- **Neon Postgres:** Backups retained 30 days (matches our policy)
- **LiveKit:** Session data deleted within 7 days after stream ends
- **Stripe:** Transaction data retained 7 years (financial regulations)
- **Cloudflare R2:** Objects deleted immediately upon API request; CDN cache purged within 48 hours
- **MapTiler:** No user data retained (stateless tile delivery)

### 6.2 Processor Audits

**Frequency:** Annual audit of top 3 processors (Neon, LiveKit, Stripe)

**Verification:**
- Data deletion is executed as agreed
- Backups do not extend retention beyond policy
- Subprocessors comply with same retention requirements

**Audit Report:** Available to users upon request

## 7. Compliance Reporting

### 7.1 Internal Reporting

**Monthly Report:**
- Data retention metrics (counts per category and age)
- Deletion job execution summary
- Archival statistics
- User deletion requests processed

**Quarterly Report:**
- Compliance audit results
- Third-party processor audit summary
- Retention policy violations (if any) and remediation
- Backup and disaster recovery drill results

**Annual Report:**
- Full compliance review
- Policy updates and rationale
- GDPR/CCPA request statistics
- Data breach incidents (if any) and response

### 7.2 External Reporting

**Regulatory Disclosures:**
- Supervisory authority requests (GDPR Art. 58)
- Data breach notifications (within 72 hours)
- Annual transparency report (public)

**User Transparency:**
- Privacy Policy updates (email notification)
- Data retention metrics published in transparency report
- GDPR/CCPA request response times and fulfillment rates

## 8. Policy Updates

### 8.1 Review Schedule
This Data Retention Policy is reviewed:
- **Annually:** Full policy review and update
- **Ad Hoc:** When regulations change or new data categories are introduced
- **Trigger Events:** Data breach, regulatory inquiry, major platform feature changes

### 8.2 Change Notification
Material changes to retention periods:
- **Notice:** 30 days via email and in-app notification
- **Opt-Out:** Users may request immediate deletion before new policy takes effect
- **Effective Date:** Changes take effect 30 days after notice

### 8.3 Version History
- **v1.0 (Feb 1, 2026):** Initial policy
- Previous versions archived and available upon request

## 9. Contact Information

For data retention inquiries:
- **General Questions:** privacy@subcults.org
- **Deletion Requests:** privacy@subcults.org (Subject: "Deletion Request")
- **GDPR Requests:** dpo@subcults.org
- **Archive Access:** privacy@subcults.org (Subject: "Archive Access Request")

**Mailing Address:**  
Subcults Data Retention Office  
548 Market Street, PMB 12345  
San Francisco, CA 94104  
United States

**Response Time:** 7 business days for general inquiries, 30 days for GDPR requests

---

## 10. Additional Resources

- **[Privacy Policy](./PRIVACY_POLICY.md)** - Comprehensive privacy practices
- **[GDPR Compliance Guide](./GDPR_COMPLIANCE_GUIDE.md)** - EU data protection rights and procedures
- **[Terms of Service](./TERMS_OF_SERVICE.md)** - Usage restrictions and liability
- **[Technical Privacy Overview](../PRIVACY.md)** - Implementation details for developers

---

**Effective Date:** February 1, 2026  
**Last Updated:** February 1, 2026

*This Data Retention Policy establishes clear retention periods and deletion procedures to ensure compliance with data protection regulations. It should be reviewed by legal counsel and data protection specialists before production deployment.*
