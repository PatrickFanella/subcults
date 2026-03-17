# GDPR Compliance Guide

**For:** European Economic Area (EEA) Users & Data Subjects  
**Effective Date:** February 1, 2026  
**Last Updated:** February 1, 2026

## Introduction

This guide explains how Subcults complies with the General Data Protection Regulation (GDPR) and how you can exercise your data protection rights. This document is specifically for users in the European Economic Area (EEA), United Kingdom, and Switzerland.

**GDPR Overview:** The GDPR grants you significant control over your personal data, including the right to access, correct, delete, and transfer your information.

## 1. Our Role Under GDPR

### 1.1 Data Controller
Subcults acts as a **Data Controller** for:
- User account data (DID, profile information)
- Scene and event data you create
- Access logs and security audit trails
- Transaction metadata (amounts, dates, scene references)

**Responsibility:** We determine how and why your personal data is processed.

### 1.2 Data Processors
We use third-party **Data Processors** for specific services:
- **Neon (Postgres):** Database hosting and storage
- **LiveKit:** Live audio streaming infrastructure
- **Stripe Connect:** Payment processing (Stripe also acts as a controller for financial data)
- **Cloudflare R2:** Media storage
- **MapTiler:** Map tile delivery

**Data Processing Agreements (DPAs):** We have signed DPAs with all processors to ensure GDPR compliance.

## 2. Lawful Basis for Processing

Under GDPR Article 6, we process your personal data based on the following lawful bases:

| Data Category | Lawful Basis | Explanation |
|---------------|--------------|-------------|
| **Account Data** | Contract Performance (Art. 6.1.b) | Necessary to provide platform services |
| **Location (Coarse)** | Contract Performance | Necessary for map-based discovery |
| **Location (Precise)** | Consent (Art. 6.1.a) | Opt-in via `allow_precise` flag |
| **Media Uploads** | Contract Performance | Necessary to display scenes and events |
| **Transaction Data** | Legal Obligation (Art. 6.1.c) | Tax and financial record-keeping |
| **Access Logs** | Legitimate Interest (Art. 6.1.f) | Security, fraud prevention, abuse detection |
| **Session Replay** | Consent (Art. 6.1.a) | Opt-in via Settings → Privacy → Session Replay |
| **Telemetry/Analytics** | Consent (Art. 6.1.a) | Opt-in via Settings → Privacy → Analytics (disabled by default) |

**Legitimate Interest Balancing:** For access logs, our legitimate interest in security and fraud prevention outweighs privacy concerns, as logs contain minimal personal data and are retained for only 90 days.

## 3. Your Rights Under GDPR

### 3.1 Right of Access (Art. 15)

**What:** You have the right to know what personal data we hold about you.

**How to Request:**
1. Email: privacy@subcults.org
2. Subject Line: "GDPR Access Request"
3. Include: Your DID, email address, and description of data you're requesting

**Response Time:** Within 30 days (1 month)

**What You'll Receive:**
- Copy of your personal data in structured, machine-readable format (JSON)
- Categories of data processed
- Purposes of processing
- Recipients of your data (third-party processors)
- Retention periods
- Information about your rights

**Cost:** First request is free; subsequent requests may incur a reasonable fee if excessive or repetitive.

### 3.2 Right to Rectification (Art. 16)

**What:** You have the right to correct inaccurate or incomplete personal data.

**How to Correct:**
1. **Self-Service:** Update your profile via Settings → Account → Edit Profile
2. **Support Request:** Email privacy@subcults.org with corrections

**Response Time:** Within 30 days (1 month)

**Scope:**
- Profile information (name, bio, avatar)
- Scene and event details
- Location data (coarse geohash, precise coordinates if opted-in)

### 3.3 Right to Erasure ("Right to Be Forgotten") (Art. 17)

**What:** You have the right to request deletion of your personal data.

**How to Request:**
1. Email: privacy@subcults.org
2. Subject Line: "GDPR Erasure Request"
3. Include: Your DID, email address, and confirmation of intent

**Response Time:** Within 30 days (1 month)

**What Will Be Deleted:**
- Account data (DID, profile, credentials)
- Scenes, events, posts you created
- Media uploads (images, audio, video)
- Alliance and trust graph data
- Access logs and audit trails (after retention period)

**Exceptions (We May Refuse Deletion If):**
- Required by law (e.g., transaction records for tax compliance - retained 7 years)
- Necessary for legal claims or defense
- Public interest or scientific research (anonymized data only)

**Soft Delete Grace Period:** 30 days before permanent deletion (allows accidental recovery)

### 3.4 Right to Restriction of Processing (Art. 18)

**What:** You can limit how we process your data while we investigate a dispute.

**How to Request:**
1. Email: privacy@subcults.org
2. Subject Line: "GDPR Restriction Request"
3. Include: Your DID, reason for restriction (e.g., accuracy dispute, objection to processing)

**Effect:**
- We will store your data but not actively process it
- You will not be able to use certain platform features
- We may still process data for legal claims or with your consent

**Duration:** Until dispute is resolved

### 3.5 Right to Data Portability (Art. 20)

**What:** You can receive your personal data in a structured, machine-readable format and transfer it to another service.

**How to Request:**
1. Email: privacy@subcults.org
2. Subject Line: "GDPR Portability Request"
3. Include: Your DID, desired format (JSON recommended)

**Response Time:** Within 30 days (1 month)

**Data Included:**
- Account and profile data
- Scenes, events, posts (with metadata)
- Alliance relationships
- Media upload references (URLs)
- Transaction history (metadata only, not payment credentials)

**Format:** JSON export with AT Protocol compatibility (where applicable)

**Note:** Media files can be downloaded separately via provided URLs.

### 3.6 Right to Object (Art. 21)

**What:** You can object to processing based on legitimate interests or for direct marketing.

**Legitimate Interest Processing:**
- **Access Logs:** You can object to processing for security/fraud prevention (we may refuse if we have compelling grounds)
- **Telemetry:** Opt-in only; if enabled, you can disable it via Settings → Privacy → Analytics (immediate effect)

**Direct Marketing:**
- **We Do Not:** Subcults does not use your data for direct marketing
- **Third Parties:** We do not sell or share your data with third-party marketers

**How to Object:**
1. Email: privacy@subcults.org
2. Subject Line: "GDPR Objection"
3. Include: Your DID, specific processing activity you're objecting to, and reasons

**Response Time:** Within 30 days (1 month)

### 3.7 Right to Withdraw Consent (Art. 7.3)

**What:** You can withdraw consent for processing at any time (without affecting prior lawful processing).

**How to Withdraw:**

#### Location Precision Consent
- **Action:** Toggle `allow_precise` to OFF for scenes/events
- **Effect:** Precise coordinates cleared immediately, only coarse geohash retained

#### Session Replay Consent
- **Action:** Settings → Privacy → Session Replay → OFF
- **Effect:** No future session data recorded; existing data deleted within 7 days

#### Telemetry/Analytics Consent
- **Action:** Settings → Privacy → Analytics → Enable
- **Default:** OFF (disabled by default)
- **Effect:** Analytics events collected only when opted-in; existing data anonymized within 90 days after opt-out

**Email Alternative:** Contact privacy@subcults.org with "Consent Withdrawal" in subject line.

### 3.8 Rights Related to Automated Decision-Making (Art. 22)

**Subcults Does NOT:**
- Make solely automated decisions with legal or significant effects
- Use automated profiling to determine access to services
- Deny services based solely on algorithmic scoring

**Trust Ranking:** Our trust-based ranking uses a transparent algorithm that weights results but does NOT:
- Deny access to content
- Automatically ban or restrict accounts
- Make legally significant decisions without human review

## 4. Data Subject Request Procedures

### 4.1 Verification Process
To protect against fraudulent requests, we verify identity before processing data subject requests:

**Step 1: Submit Request**
- Email privacy@subcults.org with request type in subject line
- Include your DID and registered email address

**Step 2: Identity Verification**
We will send a verification email to your registered address containing:
- Unique verification token
- Instructions to complete verification via AT Protocol authentication

**Step 3: Processing**
Once verified, we process your request within 30 days (1 month)

**Complex Requests:** If your request is unusually complex, we may extend the deadline by 2 additional months (we will notify you within the first month).

### 4.2 Request Tracking
All data subject requests are tracked with:
- Unique request ID
- Submission date and time
- Verification status
- Processing status (pending, in progress, completed)
- Completion date

**Status Updates:** We will email you at key milestones (verification, processing start, completion).

### 4.3 Appeals Process
If you're not satisfied with our response:
1. Email privacy@subcults.org with "GDPR Appeal" in subject line
2. Include original request ID and reason for appeal
3. We will review and respond within 30 days

If still unsatisfied, you have the right to lodge a complaint with your supervisory authority (see Section 6).

## 5. Data Transfers & International Processing

### 5.1 Where Your Data Is Processed
Your data may be processed in:
- **United States:** Primary database (Neon), payment processor (Stripe), media storage (Cloudflare R2)
- **European Economic Area:** Certain data may be cached in EU regions for performance
- **Global CDN:** MapTiler tiles delivered via global content delivery network

### 5.2 Transfer Mechanisms
For data transfers outside the EEA, we rely on:
- **Standard Contractual Clauses (SCCs):** Approved by the European Commission (Art. 46.2.c)
- **Adequacy Decisions:** For countries deemed to have adequate data protection (Art. 45)
- **Third-Party Certifications:** Some processors hold EU-U.S. Data Privacy Framework certification

### 5.3 Safeguards
All data transfers include:
- Contractual commitments to GDPR compliance
- Technical measures (encryption in transit and at rest)
- Audit rights to verify compliance
- Prompt notification of data breaches

**DPA Availability:** Copies of our Data Processing Agreements with third-party processors are available upon request.

## 6. Supervisory Authority & Complaints

### 6.1 Data Protection Officer (DPO)
For GDPR-specific inquiries, contact our Data Protection Officer:
- **Email:** dpo@subcults.org
- **Response Time:** Within 30 days

### 6.2 Right to Lodge a Complaint
If you believe we have violated GDPR, you have the right to lodge a complaint with a supervisory authority.

**Your Supervisory Authority:**
Find your local data protection authority at:
[https://edpb.europa.eu/about-edpb/board/members_en](https://edpb.europa.eu/about-edpb/board/members_en)

**Example Authorities:**
- **Germany:** Bundesbeauftragte für den Datenschutz und die Informationsfreiheit (BfDI)
- **France:** Commission Nationale de l'Informatique et des Libertés (CNIL)
- **UK:** Information Commissioner's Office (ICO)
- **Ireland:** Data Protection Commission (DPC)

**When to Complain:**
- We have not responded to your data subject request within 30 days (plus any extensions)
- You disagree with our response to your request
- You believe we are processing your data unlawfully

### 6.3 Complaint Process
1. **First:** Attempt to resolve the issue directly with us (email privacy@subcults.org)
2. **Then:** If unresolved, lodge a complaint with your supervisory authority
3. **Provide:** Copy of our response (if any), details of the issue, and supporting evidence

## 7. Data Breach Notification

In the event of a personal data breach:

### 7.1 Notification to Supervisory Authority
- **Timing:** Within 72 hours of becoming aware of the breach (Art. 33)
- **Content:** Nature of breach, categories/number of data subjects affected, likely consequences, measures taken

### 7.2 Notification to Data Subjects
- **Timing:** Without undue delay (Art. 34)
- **Trigger:** If breach is likely to result in high risk to your rights and freedoms
- **Method:** Email to registered address, plus in-app notification
- **Content:** Nature of breach, DPO contact, likely consequences, measures taken, recommendations for mitigation

**Example Scenarios Requiring Notification:**
- Unauthorized access to precise location data
- Exposure of unencrypted payment information
- Large-scale account credential leak

## 8. Children's Data

### 8.1 Age Requirement
Under GDPR Article 8, children under 16 require parental consent for data processing (or 13 in certain EU member states).

**Subcults Policy:**
- Minimum age: 16 in the EU (aligned with strictest GDPR requirement)
- Minimum age: 13 in non-EU jurisdictions
- We do not knowingly process data of children under these ages

### 8.2 Parental Rights
If you are a parent/guardian and believe your child has provided data without consent:
1. Email: privacy@subcults.org with "Child Data Removal" in subject line
2. We will delete the account and associated data within 72 hours
3. No verification required if child is clearly under minimum age

## 9. GDPR-Compliant Features

### 9.1 Privacy by Design
Our platform implements GDPR's "privacy by design" principle (Art. 25):
- **Default Privacy:** `allow_precise = FALSE` for location data
- **EXIF Stripping:** Automatic metadata removal from media uploads
- **PII Redaction:** Client error logs sanitized before transmission
- **Minimal Data Collection:** Only data necessary for platform operation

### 9.2 Privacy by Default
Default settings prioritize privacy:
- Session replay: OFF
- Telemetry: OFF (user must opt-in)
- Scene visibility: PUBLIC (but location is jittered)
- Alliance visibility: Controlled by scene organizers

### 9.3 Data Minimization
We collect only data necessary for:
- Providing platform services (account, scenes, events, streaming)
- Security and fraud prevention (access logs)
- Legal compliance (transaction records)

**We Do Not Collect:**
- IP addresses (except transiently for rate limiting)
- Browsing history or behavioral tracking
- Full URLs with query parameters
- Unnecessary personal identifiers

## 10. Technical Implementation

### 10.1 Consent Management
All consent-based processing is implemented with:
- **Explicit Opt-In:** For precise location, session replay
- **Granular Controls:** Separate toggles for each data category
- **Easy Withdrawal:** One-click opt-out in Settings
- **Audit Trail:** Consent changes logged with timestamps

### 10.2 Data Export Format
Data portability exports use:
- **JSON format:** Machine-readable and human-readable
- **AT Protocol compatibility:** DIDs and record schemas
- **Comprehensive metadata:** Creation dates, update dates, relationships
- **Media references:** URLs to download full files separately

### 10.3 Deletion Implementation
Right to erasure is implemented with:
- **Soft Delete:** 30-day grace period for accidental recovery
- **Hard Delete:** Permanent deletion after grace period
- **Cascading Deletion:** Related data (alliances, memberships, media) also deleted
- **Audit Trail:** Deletion events logged (anonymized after 90 days)

**Exception:** Transaction records retained 7 years for legal compliance (anonymized where possible).

## 11. Updates to This Guide

This GDPR Compliance Guide is reviewed and updated:
- **Annually:** As part of privacy policy review cycle
- **Ad Hoc:** When GDPR guidance or regulations change
- **Notification:** Material changes communicated via email

**Version History:** Previous versions archived and available upon request.

## 12. Contact Information

### 12.1 Data Subject Requests
- **Email:** privacy@subcults.org
- **Subject Line:** "GDPR [Request Type]" (e.g., "GDPR Access Request")

### 12.2 Data Protection Officer
- **Email:** dpo@subcults.org
- **Response Time:** Within 30 days

### 12.3 General Privacy Inquiries
- **Email:** privacy@subcults.org
- **Response Time:** Within 7 business days

### 12.4 Mailing Address
Subcults Data Protection Office  
548 Market Street, PMB 12345  
San Francisco, CA 94104  
United States

---

## 13. Additional Resources

- **[Privacy Policy](./PRIVACY_POLICY.md)** - Comprehensive privacy practices
- **[Terms of Service](./TERMS_OF_SERVICE.md)** - Usage restrictions and liability
- **[Data Retention Policy](./DATA_RETENTION_POLICY.md)** - Data lifecycle and deletion procedures
- **[Technical Privacy Overview](../PRIVACY.md)** - Implementation details for developers

---

**Effective Date:** February 1, 2026  
**Last Updated:** February 1, 2026

*This GDPR Compliance Guide is intended to help users understand and exercise their data protection rights. It should be reviewed by legal counsel specializing in EU data protection law before production deployment.*
