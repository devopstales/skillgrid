# Product Requirements: [Product Name]

> Create this as `.skillgrid/artifacts/00-prd.md`. It is the reference for
> scope, features, and target state. The one-paragraph product statement, the
> VERIFIED facts, the in-force ADR index, and the locked constraints live in
> `.skillgrid/ASSUMPTIONS.md` (see `templates/ASSUMPTIONS.md`), which links
> here. Where this file and `ASSUMPTIONS.md` § VERIFIED disagree, VERIFIED wins.
> For a multi-project repo, merge rather than overwrite.

## Product Overview

**Product Vision:** [1-2 sentence description of what this product is and why it exists]

**Target Users:** [Primary and secondary user personas]

**Business Objectives:** [Key business goals this product aims to achieve]

**Success Metrics:** [How success will be measured — specific and measurable]

## User Personas

### Persona 1: [Name]
- **Demographics:** [Age, occupation, technical proficiency]
- **Goals:** [What they want to accomplish]
- **Pain Points:** [Current challenges they face]
- **User Journey:** [How they'll interact with this product]

### Persona 2: [Name]
- **Demographics:** [Age, occupation, technical proficiency]
- **Goals:** [What they want to accomplish]
- **Pain Points:** [Current challenges they face]
- **User Journey:** [How they'll interact with this product]

## Feature Requirements

Prioritize with MoSCoW: **M**ust-have / **S**hould-have / **C**ould-have / **W**on't-have (this release).

| Feature | Description | User Stories | Priority | Acceptance Criteria | Dependencies |
|---------|-------------|--------------|----------|---------------------|--------------|
| **[Feature 1]** | [Brief description] | [As a user, I want to...] | [Must/Should/Could/Won't] | [List of criteria] | [Dependencies] |
| **[Feature 2]** | [Brief description] | [As a user, I want to...] | [Must/Should/Could/Won't] | [List of criteria] | [Dependencies] |
| **[Feature 3]** | [Brief description] | [As a user, I want to...] | [Must/Should/Could/Won't] | [List of criteria] | [Dependencies] |

## User Flows

### Flow 1: [Name, e.g., User Registration]
1. [Step 1]
2. [Step 2]
3. [Step 3]
   - [Alternative path]
   - [Error state]

### Flow 2: [Name]
1. [Step 1]
2. [Step 2]
3. [Step 3]
   - [Alternative path]
   - [Error state]

## Non-Functional Requirements

### Performance
- **Load Time:** [Target load time]
- **Concurrent Users:** [Expected number]
- **Response Time:** [Target response time]

### Security
- **Authentication:** [Requirements]
- **Authorization:** [User permission levels]
- **Data Protection:** [Requirements]

### Compatibility
- **Devices:** [Supported devices]
- **Browsers:** [Supported browsers and versions]
- **Screen Sizes:** [Supported dimensions]

### Accessibility
- **Compliance Level:** [e.g., WCAG 2.1 AA]
- **Specific Requirements:** [Key accessibility features]

## Technical Specifications

### Frontend
- **Technology Stack:** [Framework, libraries]
- **Design System:** [Design system to use]
- **Responsive Design:** [Requirements]

### Backend
- **Technology Stack:** [Languages, frameworks]
- **API Requirements:** [RESTful, GraphQL, etc.]
- **Database:** [Database type and structure]

### Infrastructure
- **Hosting:** [Hosting solutions]
- **Scaling:** [Scaling requirements]
- **CI/CD:** [Deployment process]

## Analytics & Monitoring

- **Key Metrics:** [Metrics to track]
- **Events:** [User events to capture]
- **Dashboards:** [Required dashboards]
- **Alerting:** [Alert thresholds]

## Release Planning

### MVP (v1.0)
- **Features:** [List of MVP features]
- **Timeline:** [Expected release date]
- **Success Criteria:** [How to measure MVP success]

### Future Releases
- **v1.1:** [Feature set and expected timeline]
- **v1.2:** [Feature set and expected timeline]
- **v2.0:** [Feature set and expected timeline]

## Appendix

### Competitive Analysis
- **Competitor 1:** [Strengths and weaknesses]
- **Competitor 2:** [Strengths and weaknesses]

### User Research Findings
- **Finding 1:** [Key insight from research]
- **Finding 2:** [Key insight from research]

### Glossary
- Terms live in `.skillgrid/artifacts/01-business-terms.md` and `02-technical-terms.md`.
