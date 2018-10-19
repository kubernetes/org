---
name: Repository creation/migration
about: Create or migrate a repository into a Kubernetes Org

---

### New Repo, Staging Repo, or migrate existing
e.g. new repository

### Requested name for new repository
e.g. cloud-provider-foo

### Which Organization should it reside
e.g. kubernetes-sigs

### If not a staging repo, who should have admin access
e.g. alice, bob

### If not a staging repo, who should have write access
e.g. chris, dianne

### If a new repo, who should be listed as approvers in OWNERS
e.g. alice, bob

### If a new repo, who should be listed in SECURITY_CONTACTS
e.g. alice, bob

### What should the repo description be
by default we will set it to "created due to (link to this issue)"

### What SIG and subproject does this fall under in sigs.yaml
e.g. this is a new subproject for sig-foo called bar-baz
e.g. this is part of the sparkles subproject for sig-awesome

### Approvals
Please prove you have followed the appropriate approval process for this new
repo by including links to the relevant approvals (meeting minutes, e-mail
thread, etc.)

Authoritative requirements are here: https://git.k8s.io/community/github-management/kubernetes-repositories.md

tl;dr (but really you should read the linked doc, this may be stale)
- If this is a core repository, then sig-architecture must approve
- If this is a SIG repository, then this must follow the procedure spelled out
  in that SIG's charter

### Additional context for request
Any additional information or context to describe the request.

<!-- DO NOT EDIT BELOW THIS LINE -->
/area github-repo
