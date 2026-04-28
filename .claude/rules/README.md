# Steins Development Rules

This directory contains the comprehensive ruleset for developing **Steins**, a Go-based manga viewing platform. These rules guide the implementation of a scalable, extensible content delivery and reading service.

## Overview

**Steins** is a service designed to:
- Serve a catalog of manga series, chapters, and pages to readers
- Provide a smooth web reading experience (paged, vertical-scroll, RTL/LTR)
- Manage user accounts, libraries, bookmarks, and reading progress
- Handle image upload, optimization, and CDN-friendly delivery
- Support search, genre/tag browsing, and recommendations

## Ruleset Structure

The rules are organized into specialized domains:

### [01-architecture.md](01-architecture.md)
**Core System Design and Architecture**

Covers:
- Overall system architecture and layers
- Technology stack requirements
- Directory structure (Standard Go Project Layout)
- Request/job data flow
- Scalability considerations
- Multi-environment configuration

Read this first to understand:
- How components fit together (HTTP API, services, workers, storage)
- Design principles and patterns
- Infrastructure requirements
- Storage strategy (relational + object + cache)

### [02-api-implementation.md](02-api-implementation.md)
**HTTP API and Reader Service Standards**

Covers:
- HTTP API design (REST conventions, versioning)
- Authentication & authorization (JWT, sessions, OAuth2)
- Manga / Chapter / Page resource models
- Reader endpoints (page streaming, progress tracking)
- Rate limiting and request validation
- Pagination, filtering, sorting
- Image delivery and signed URLs

Essential for:
- Implementing public-facing endpoints
- Adding new resources or query parameters
- Securing reader and library endpoints
- Integrating with the CDN / object storage

### [04-error-handling.md](04-error-handling.md)
**Error Handling, Monitoring, and Observability**

Covers:
- Error taxonomy and custom error types
- Retry policies and circuit breakers
- Structured logging standards
- Metrics collection with Prometheus
- Health checks and alerting
- Incident response procedures
- Data integrity checks

Critical for:
- Production reliability
- Debugging and troubleshooting
- Performance monitoring
- Operational excellence

### [05-testing.md](05-testing.md)
**Testing Strategy and Quality Assurance**

Covers:
- Unit, integration, and E2E testing
- Mocking strategies
- Table-driven tests
- Benchmarking and profiling
- Code coverage requirements
- Linting and quality gates
- CI/CD workflows
- Load testing

Important for:
- Ensuring code quality
- Preventing regressions
- Performance optimization
- Maintaining test coverage

### [06-code-style.md](06-code-style.md)
**Code Style and Conventions**

Covers:
- Go formatting standards (tabs via gofmt)
- Naming conventions
- Error handling patterns
- Function and struct design
- Comments and documentation
- Database and configuration styles
- Testing conventions
- Anti-patterns to avoid

Essential for:
- Consistent codebase
- Code readability
- Team collaboration
- Code review efficiency

## Quick Start Guide

### For New Developers

1. **Start Here**: Read [01-architecture.md](01-architecture.md) for system overview
2. **Code Standards**: Read [06-code-style.md](06-code-style.md) for style guidelines
3. **Set Up Environment**: Follow technology stack requirements
4. **Understand Data Flow**: Review the request and job pipelines
5. **Reference Rules**: Use relevant sections while coding

### For Specific Tasks

**Adding a New API Endpoint:**
1. Review [02-api-implementation.md](02-api-implementation.md) - Resource Conventions
2. Implement handler + service + repository following the layering rules
3. Add tests per [05-testing.md](05-testing.md)
4. Configure error handling per [04-error-handling.md](04-error-handling.md)

**Debugging Production Issues:**
1. Check [04-error-handling.md](04-error-handling.md) - Incident Response
2. Review metrics and logs
3. Follow debugging procedures
4. Update runbooks if needed

**Optimizing Performance:**
1. Review [01-architecture.md](01-architecture.md) - Caching Strategy
2. Run benchmarks per [05-testing.md](05-testing.md)
3. Check resource usage in monitoring
4. Implement optimizations incrementally

## Key Principles

### 1. Extensibility First
- Design for adding new content types and image formats easily
- Use interface-based architecture
- Plugin-friendly storage and search backends
- Configuration-driven behavior

### 2. Reader Experience
- Image delivery must be fast and CDN-friendly
- Reader state (progress, bookmarks) must be reliable
- Graceful degradation on slow networks
- Mobile-first response sizes

### 3. Reliability
- Handle failures gracefully
- Implement comprehensive error handling
- Retry with backoff for transient failures
- Monitor and alert proactively

### 4. Performance
- Design for horizontal scaling
- Cache hot resources (popular series, recent chapters)
- Optimize hot paths (page reads, list queries)
- Batch and async where appropriate

### 5. Maintainability
- Write clear, documented code
- Follow Go best practices
- Comprehensive testing
- Keep dependencies minimal

## Development Workflow

### Before Writing Code

1. **Understand the Requirement**
   - What problem are we solving?
   - Which component does this affect (API, worker, storage)?
   - Are there existing patterns to follow?

2. **Review Relevant Rules**
   - Check the appropriate ruleset section
   - Understand interfaces and patterns
   - Review error handling requirements
   - Plan test coverage

3. **Design Before Implementation**
   - Sketch request/data flow
   - Identify dependencies
   - Plan for failure cases
   - Consider performance impact

### While Writing Code

1. **Follow the Style Guide** ([06-code-style.md](06-code-style.md))
   - gofmt-style indentation (tabs)
   - Clear, self-documenting names
   - Minimal comments (only WHY)
   - Early returns for errors

2. **Follow the Architecture Rules**
   - Use prescribed interfaces and layering
   - Implement proper error handling
   - Add structured logging
   - Include context in operations

3. **Write Tests Alongside**
   - Unit tests for logic
   - Integration tests for handlers/repositories
   - Test error paths
   - Add benchmarks for critical paths

4. **Keep it Simple**
   - Remove unnecessary code
   - Avoid premature abstraction
   - Delete commented-out code
   - One responsibility per function

### After Writing Code

1. **Self-Review** (Use [06-code-style.md](06-code-style.md) checklist)
   - Does it follow the style guide?
   - Does it follow the architecture?
   - Are all error cases handled?
   - Is logging sufficient?
   - Are tests comprehensive?
   - No commented-out code?
   - No magic numbers?

2. **Quality Checks**
   - Run linters (golangci-lint)
   - Check test coverage (≥70%)
   - Run benchmarks
   - Verify no new warnings
   - Check code formatting

3. **Integration Verification**
   - Test against a real PostgreSQL/Redis (dev environment)
   - Check metrics and logs
   - Verify monitoring/alerting
   - Document any new configurations

## Common Patterns

### HTTP Handler

```go
// Follow handler conventions from 02-api-implementation.md
type ChapterHandler struct {
	service ChapterService
}

func (h *ChapterHandler) GetChapter(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	chapter, err := h.service.GetByID(ctx, id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, chapter)
}
```

### Service Layer

```go
// Service orchestrates repositories and external systems
type ChapterService struct {
	chapters ChapterRepository
	pages    PageRepository
	storage  ObjectStorage
}

func (s *ChapterService) GetByID(ctx context.Context, id string) (*Chapter, error) {
	chapter, err := s.chapters.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find chapter %s: %w", id, err)
	}

	pages, err := s.pages.ListByChapter(ctx, chapter.ID)
	if err != nil {
		return nil, fmt.Errorf("list pages for chapter %s: %w", chapter.ID, err)
	}

	chapter.Pages = pages
	return chapter, nil
}
```

### Error Handling

```go
// Follow error patterns from 04-error-handling.md
func (s *ChapterService) GetByID(ctx context.Context, id string) (*Chapter, error) {
	chapter, err := s.chapters.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, NewNotFoundError(CodeChapterNotFound, "chapter not found", id)
		}
		return nil, NewStorageError(CodeStorageRead, "find chapter", err)
	}
	return chapter, nil
}
```

## Integration with Claude

When working with Claude for development:

1. **Reference Specific Rules**
   - "Following the API conventions from 02-api-implementation.md..."
   - "As per the error handling rules in 04-error-handling.md..."

2. **Ask for Clarification**
   - "Which pattern from the ruleset should I follow here?"
   - "How does this fit into the architecture from 01-architecture.md?"

3. **Validate Against Rules**
   - "Does this implementation follow the content pipeline rules?"
   - "Is this error handling consistent with the ruleset?"

## Updates and Evolution

These rules are living documents and should evolve with the project:

### When to Update Rules

- New patterns emerge across the codebase
- Best practices are discovered
- Performance optimizations are proven
- New technologies are adopted
- Production issues reveal gaps

### How to Update

1. Propose changes via discussion
2. Update relevant ruleset file
3. Update this README if structure changes
4. Communicate changes to team
5. Update code to follow new rules

## Additional Resources

### External Documentation

- Go Best Practices: https://go.dev/doc/effective_go
- Prometheus Best Practices: https://prometheus.io/docs/practices/
- PostgreSQL Performance: https://wiki.postgresql.org/wiki/Performance_Optimization
- Asynq (background jobs): https://github.com/hibiken/asynq

### Internal Documentation

- API Documentation: `docs/api/`
- Deployment Guide: `docs/deployment/`
- Runbooks: `docs/runbooks/`
- Architecture Diagrams: `docs/architecture/`

## Getting Help

- Review the relevant ruleset section first
- Check existing code for examples
- Ask in team chat with specific questions
- Reference the ruleset section in your question

---

**Remember**: These rules exist to ensure consistency, quality, and maintainability. When in doubt, follow the rules. If the rules don't cover a scenario, that's an opportunity to improve them.
