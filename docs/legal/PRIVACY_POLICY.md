# Privacy Policy

**Effective Date:** February 1, 2026  
**Last Updated:** February 1, 2026

## Introduction

Subcults ("we," "us," or "our") is committed to protecting the privacy of underground music communities. This Privacy Policy describes how we collect, use, store, and protect your personal information when you use our platform.

**Our Core Principle:** Privacy first. We believe in presence over popularity, and that means respecting your privacy, protecting your location data, and giving you control over your information.

## 1. Information We Collect

### 1.1 Information You Provide Directly

#### Account Information
- **Decentralized Identifier (DID):** We use AT Protocol DIDs for authentication, enabling data portability and reducing platform lock-in
- **Profile Information:** Display name, bio, avatar image, and other optional profile details
- **Scene & Event Data:** Information about scenes you create, events you organize, posts you publish

#### Location Information
- **Coarse Location:** 6-character geohash (~±0.61 km accuracy) for regional discovery without pinpointing exact venues
- **Precise Location (Opt-In Only):** Exact GPS coordinates are **only** stored when you explicitly enable `allow_precise` for a scene or event
- **Default Setting:** `allow_precise = FALSE` — we never store precise coordinates without your explicit consent

#### Media Uploads
- **Images, Audio, Video:** Files you upload for scenes, events, posts, or profile images
- **EXIF Stripping:** All uploaded media has metadata automatically removed before storage to prevent location/device leakage:
  - GPS coordinates embedded in photos
  - Device identifiers (camera make, model, serial numbers)
  - Original timestamps
  - Camera settings metadata

#### Payment Information
- **Stripe Connect:** When you onboard as an artist/organizer, Stripe processes your banking and identity information directly
- **Transaction Data:** We store transaction metadata (amounts, dates, scene/event references) but never store full payment credentials

### 1.2 Information Collected Automatically

#### Usage Data
- **Request Logs:** HTTP method, path (no query parameters), status code, latency, response size
- **Request IDs:** UUID correlation identifiers for debugging (from `X-Request-ID` header or auto-generated)
- **Authentication Context:** Your DID when authenticated (for access control and audit trails)

#### Error & Diagnostic Data
- **Client-Side Errors:** Error messages, stack traces, component traces, URL, user agent, session ID
  - **PII Redaction:** JWT tokens, email addresses, DIDs, authorization headers, and API keys are automatically removed before transmission
  - **Rate Limited:** Maximum 10 errors per minute to prevent abuse
- **Session Replay (Opt-In Only):** 
  - **Default:** OFF (you must explicitly enable)
  - **Data Recorded:** DOM changes (sampled 50%), clicks (100%), navigation (100%), scroll (10%)
  - **Privacy Protection:** No text content, no form values, element IDs/classes only, pathname only (no query params)
  - **Transmission:** Only sent when an error occurs AND you have opted in

#### Performance & Analytics (Opt-In)
- **Telemetry Events:** Usage analytics, performance metrics, feature engagement
- **Default:** OFF (disabled by default - user must opt-in)
- **User Control:** Can be enabled in Settings → Privacy → Analytics
- **Scope:** Aggregated data not tied to individual users after 90 days

### 1.3 Information We Do NOT Collect

We explicitly **do not** log or store:
- IP addresses (except transiently for rate limiting decisions)
- Request bodies or form data
- Full URLs with query parameters
- Authentication credentials or tokens
- User movements or browsing patterns

## 2. How We Use Your Information

### 2.1 Core Platform Functions
- **Identity & Authentication:** Verifying your DID and managing access tokens
- **Scene Discovery:** Displaying scenes and events on the map based on location and trust graph
- **Live Streaming:** Connecting you to LiveKit audio rooms for live performances
- **Payments:** Processing transactions via Stripe Connect for tickets, merchandise, and direct artist support
- **Media Storage:** Storing your uploaded content in Cloudflare R2

### 2.2 Trust & Ranking
- **Alliance System:** Computing trust scores based on your alliances and role multipliers
- **Search Ranking:** Weighting search results by composite score (text relevance, proximity, recency, trust weight when feature-flagged)

### 2.3 Platform Improvement
- **Error Debugging:** Analyzing client-side error logs to identify and fix bugs
- **Performance Optimization:** Using telemetry data to improve load times and responsiveness
- **Feature Development:** Understanding feature engagement to prioritize development

### 2.4 Security & Compliance
- **Access Auditing:** Maintaining security logs for 90 days to detect abuse
- **Rate Limiting:** Preventing API abuse and ensuring fair resource allocation
- **Legal Compliance:** Responding to valid legal requests (e.g., GDPR data subject requests)

## 3. Information Sharing & Third Parties

We share information with third-party services only as necessary to operate the platform:

### 3.1 Third-Party Services

| Service | Purpose | Data Shared | Privacy Policy |
|---------|---------|-------------|----------------|
| **Neon Postgres** | Primary database | All platform data (scenes, events, users, transactions) | [Neon Privacy](https://neon.tech/privacy-policy) |
| **LiveKit Cloud** | Live audio streaming (WebRTC SFU) | Room metadata, participant identities, audio streams | [LiveKit Privacy](https://livekit.io/privacy-policy) |
| **Stripe Connect** | Payment processing | Transaction amounts, artist banking info (processed by Stripe) | [Stripe Privacy](https://stripe.com/privacy) |
| **Cloudflare R2** | Media storage | Uploaded images, audio, video files (EXIF-stripped) | [Cloudflare Privacy](https://www.cloudflare.com/privacypolicy/) |
| **MapTiler** | Map tiles | Tile requests with no user identification | [MapTiler Privacy](https://www.maptiler.com/privacy-policy/) |
| **Jetstream (AT Protocol)** | Decentralized data ingestion | Public AT Protocol records you publish | [Bluesky Privacy](https://bsky.social/about/support/privacy-policy) |

### 3.2 Data Transfers
- **U.S.-Based Services:** Most third-party services are based in the United States
- **Standard Contractual Clauses:** We rely on SCCs and adequacy decisions for international data transfers where applicable
- **No Selling of Data:** We never sell your personal information to third parties

### 3.3 Legal Disclosure
We may disclose your information if required by law, court order, or:
- To protect our legal rights or defend against legal claims
- To prevent fraud, security threats, or illegal activity
- To comply with valid data subject requests (GDPR, CCPA, etc.)

## 4. User Rights & Controls

### 4.1 Location Privacy Controls
- **Consent Management:** Toggle `allow_precise` on/off for each scene or event
- **Geographic Privacy:** Public coordinates use deterministic geohash-based jitter to prevent precise location tracking
- **Default Protection:** Precise location is NEVER stored without explicit opt-in

### 4.2 Telemetry & Diagnostics Controls
- **Session Replay:** OFF by default; toggle in Settings → Privacy → Session Replay
- **Analytics Opt-In:** Enable telemetry in Settings → Privacy → Analytics (disabled by default)
- **Error Logging:** Always active (essential for stability), but PII is automatically redacted

### 4.3 Data Access & Portability
- **View Your Data:** Request a copy of your personal data via support contact
- **Data Export:** Download your scenes, events, posts, and profile information
- **AT Protocol Portability:** Your DID-based identity enables data portability across AT Protocol platforms

### 4.4 Data Deletion
- **Account Deletion:** Request account deletion via support contact
- **Soft Delete Grace Period:** 30 days before permanent deletion (allows recovery if accidental)
- **Content Removal:** Delete individual scenes, events, or posts at any time
- **Right to Be Forgotten:** Request complete erasure of your personal data (see GDPR Compliance Guide)

### 4.5 Content Ownership
- **Your Content:** You retain all rights to content you create (scenes, events, posts, media)
- **Platform License:** By posting, you grant us a license to display and distribute your content on the platform
- **Deletion Impact:** Removing content may affect other users' experience (e.g., events you organized)

## 5. Data Retention

We retain personal information only as long as necessary for the purposes described in this policy:

| Data Type | Retention Period | Rationale |
|-----------|------------------|-----------|
| **Access Logs** | 90 days | Security audit trail, abuse detection |
| **Client Error Logs** | 30 days | Debugging and stability analysis |
| **Session Replay Data** | 7 days | Only when opted-in; auto-deleted after 7 days |
| **Soft-Deleted Content** | 30 days | Grace period before permanent deletion |
| **Session Tokens** | Until expiry | Access: 15min, Refresh: 7 days |
| **Uploaded Media** | Until user deletion | Subject to storage quotas |
| **Telemetry Events** | 90 days | Aggregated for analytics; not tied to users after 90 days |
| **Transaction Metadata** | 7 years | Legal compliance (tax, financial regulations) |
| **Account Data** | Until account deletion | Core platform functionality |

**Archival & Deletion:** See the [Data Retention Policy](./DATA_RETENTION_POLICY.md) for detailed procedures.

## 6. Security Measures

### 6.1 Technical Safeguards
- **Encryption in Transit:** All data transmitted via HTTPS/TLS
- **Encryption at Rest:** Database and media storage use provider-managed encryption
- **JWT Authentication:** Access tokens (15min expiry), refresh tokens (7 day expiry) with HS256 signing
- **Rate Limiting:** Tiered limits to prevent abuse (Global: 100/min, Auth: 10/min, Search: 30/min)
- **Input Validation:** All user input sanitized and validated
- **EXIF Stripping:** Automatic metadata removal from uploaded media

### 6.2 Access Controls
- **Scene Visibility:** Public, members-only, or hidden modes
- **Role-Based Access:** Organizers, artists, promoters, members have different permissions
- **Audit Logging:** Privacy-compliant access logs for security monitoring

### 6.3 Incident Response
In the event of a data breach:
1. We will investigate and contain the breach within 24 hours
2. Affected users will be notified within 72 hours (GDPR requirement)
3. Regulatory authorities will be notified as required by law
4. Remediation measures will be implemented immediately

## 7. Children's Privacy

Subcults is not intended for users under 13 years of age (or 16 in the EU). We do not knowingly collect personal information from children. If you believe we have collected data from a child, please contact us immediately at privacy@subcults.org.

## 8. International Users & GDPR

### 8.1 EU Data Protection Rights
If you are in the European Economic Area (EEA), you have the following rights under GDPR:
- **Right of Access:** Request a copy of your personal data
- **Right to Rectification:** Correct inaccurate or incomplete data
- **Right to Erasure:** Request deletion of your personal data ("right to be forgotten")
- **Right to Restriction:** Limit how we process your data
- **Right to Data Portability:** Receive your data in a structured, machine-readable format
- **Right to Object:** Object to processing based on legitimate interests
- **Right to Withdraw Consent:** Withdraw consent for location precision, telemetry, session replay

**How to Exercise Your Rights:** See the [GDPR Compliance Guide](./GDPR_COMPLIANCE_GUIDE.md) for detailed procedures.

### 8.2 Legal Basis for Processing
- **Contract Performance:** Processing necessary to provide the platform (scenes, events, streaming)
- **Legitimate Interests:** Security, fraud prevention, platform improvement
- **Consent:** Location precision, session replay, telemetry (all opt-in)
- **Legal Obligation:** Compliance with tax, financial, and data protection laws

### 8.3 Data Protection Officer
For GDPR-related inquiries, contact our Data Protection Officer:
- **Email:** dpo@subcults.org
- **Response Time:** Within 30 days of receipt

### 8.4 Supervisory Authority
You have the right to lodge a complaint with your local data protection authority:
- **EU Users:** [European Data Protection Board - Find Your Authority](https://edpb.europa.eu/about-edpb/board/members_en)

## 9. California Privacy Rights (CCPA)

California residents have additional rights under the California Consumer Privacy Act (CCPA):
- **Right to Know:** Request disclosure of data categories collected, sources, purposes, and third parties
- **Right to Delete:** Request deletion of personal information
- **Right to Opt-Out:** Opt out of sale of personal information (we do NOT sell data)
- **Non-Discrimination:** We will not discriminate against you for exercising your privacy rights

**How to Exercise Your Rights:** Email privacy@subcults.org with "CCPA Request" in the subject line.

## 10. Changes to This Policy

We may update this Privacy Policy periodically to reflect changes in our practices or legal requirements:
- **Notification:** We will notify users of material changes via email or in-app notification
- **Review Schedule:** This policy is reviewed and updated at least annually
- **Effective Date:** Changes take effect 30 days after notification (or immediately if required by law)
- **Version History:** Previous versions are archived and available upon request

## 11. Contact Us

For privacy-related questions, concerns, or requests:

- **General Privacy Inquiries:** privacy@subcults.org
- **Data Subject Requests:** privacy@subcults.org (Subject: "Data Request")
- **Security Issues:** security@subcults.org
- **Data Protection Officer (GDPR):** dpo@subcults.org

**Mailing Address:**  
Subcults Privacy Office  
548 Market Street, PMB 12345  
San Francisco, CA 94104  
United States

**Response Time:** We aim to respond to all privacy inquiries within 7 business days, and data subject requests within 30 days as required by GDPR.

---

## 12. Additional Resources

- **[Terms of Service](./TERMS_OF_SERVICE.md)** - Usage restrictions, liability, dispute resolution
- **[GDPR Compliance Guide](./GDPR_COMPLIANCE_GUIDE.md)** - Detailed procedures for data subject requests
- **[Data Retention Policy](./DATA_RETENTION_POLICY.md)** - Data categories, archival, and deletion procedures
- **[Technical Privacy Overview](../PRIVACY.md)** - Implementation details for developers and contributors

---

**Acknowledgment:** By using Subcults, you acknowledge that you have read and understood this Privacy Policy. If you do not agree with our practices, please do not use the platform.

*This Privacy Policy is intended for transparency and user education. It should be reviewed by legal counsel before production deployment.*
