# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

We take the security of STRATINT seriously. If you believe you have found a security vulnerability, please report it to us responsibly.

### How to Report

1. **Do NOT** open a public GitHub issue for security vulnerabilities
2. Email your findings to the repository maintainers
3. Include as much detail as possible:
   - Type of vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- We will acknowledge receipt of your vulnerability report within 48 hours
- We will provide a more detailed response within 7 days
- We will work with you to understand and resolve the issue
- We will notify you when the vulnerability has been fixed

### Security Best Practices for Deployment

When deploying STRATINT, follow these security best practices:

1. **Environment Variables**: Never commit secrets to version control. Use environment variables or secret management services.

2. **Database Security**:
   - Use strong, unique passwords
   - Enable SSL/TLS for database connections
   - Restrict database access to necessary services only

3. **API Keys**:
   - Rotate API keys regularly
   - Use separate keys for development and production
   - Store keys in secure secret management (e.g., Google Secret Manager, AWS Secrets Manager)

4. **Network Security**:
   - Deploy behind a reverse proxy with TLS
   - Use firewalls to restrict access
   - Enable rate limiting

5. **Authentication**:
   - Use strong JWT secrets (32+ random bytes)
   - Set appropriate token expiration times
   - Use secure password hashing (bcrypt)

## Security Features

STRATINT includes the following security features:

- JWT-based authentication for admin panel
- Prepared statements for all database queries (SQL injection prevention)
- Input validation and sanitization
- CORS configuration
- Secure headers middleware
- Non-root Docker container execution

## Acknowledgments

We appreciate the security research community and will acknowledge researchers who report valid vulnerabilities (with their permission).
