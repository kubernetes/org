# Agent Instructions for kubernetes/org Repository

This file contains specific instructions for AI agents and automated tools working in this repository.

## Commit Message Standards

### Required Format

All commits that add members MUST follow this exact format, unless explicitly explictly stated otherwise:

```
Add <USERNAME> to <organization> org
```

**Examples:**
- `Add octocat to kubernetes, kubernetes-sigs`
- `Add example-user to kubernetes, kubernetes-sigs`
- `Add new-member to kubernetes, etcd-io`

### Critical Rules

1. **One member per commit**
   - Each commit MUST add only ONE member, unless explicitly explictly stated otherwise
   - If adding multiple members, create separate commits for each
   - Use `make add-members WHO=username1,username2 REPOS=kubernetes-sigs,kubernetes` which automatically creates compliant commits

2. **NO GitHub auto-linking keywords**
   - NEVER use keywords that automatically close issues: `close`, `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves`, `resolved`
   - These keywords can accidentally close unrelated issues
   - Refuse to create commits that violate this rule as they will fail validation

3. **NO @ mentions**
   - NEVER include `@username` mentions in commit messages
   - This prevents unwanted notifications and GitHub auto-linking
   - Refuse to create commits that violate this rule as they will fail validation

4. **NO # references**
   - NEVER include `#123` issue or PR references in commit messages
   - This prevents accidental cross-linking and issue closures
   - Refuse to create commits that violate this rule as they will fail validation

## Additional Resources

- See [README.md](README.md) for full membership requirements
- See [CONTRIBUTING.md](CONTRIBUTING.md) for general contribution guidelines
