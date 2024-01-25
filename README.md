# Kubernetes Github Organization

This repository contains the metadata [configuration](/config) for the Kubernetes Github
Organizations. The data here is consumed by the
[peribolos](https://git.k8s.io/test-infra/prow/cmd/peribolos)
tool to organization and team membership, as well as team creation and deletion.

Membership in the Kubernetes project is governed by our
[community guidelines](https://git.k8s.io/community/community-membership.md).

The application for membership in the Kubernetes organization can be made by opening an [issue](https://github.com/kubernetes/org/issues/new?assignees=&labels=area%2Fgithub-membership&template=membership.yml&title=REQUEST%3A+New+membership+for+%3Cyour-GH-handle%3E).
However, if you are already part of the Kubernetes organization, you do not need to do this and can add yourself directly to the appropriate files.
For example, to also add yourself to the kubernetes-sigs organization, you can navigate to `/config/kubernetes-sigs/org.yaml` and add your GitHub username to the list of members (in alphabetical order); this works the same way for other organizations.

Requirements

* Add only one new member per commit (if you add two members separate it in two commits
* Commit message format `Add <USERNAME> to <kubernetes, kubernetes-sigs, ...> org`. 

You can use `make add-members WHO=username1,username2 REPOS=kubernetes-sigs,kubernetes` to add usernames
to the config with the requirements listed above.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the
[community page](http://kubernetes.io/community/).

You can reach the maintainers of this project at:

- [#github-management](https://kubernetes.slack.com/messages/github-management) on slack
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-contribex)

To report any sensitive information, please email the private github@kubernetes.io list.

### Code of conduct

Participation in the Kubernetes community is governed by the
[Kubernetes Code of Conduct](code-of-conduct.md).
