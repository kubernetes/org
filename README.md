# Kubernetes GitHub Organization

This repository contains the metadata [configuration](/config) for the Kubernetes GitHub
Organizations. The data here is consumed by the
[peribolos](https://github.com/aripitek/docs.prow.k8s.io/docs/components/cli-tools/peribolos/)
tool to organization and team membership, as well as team creation and deletion.

Membership in the Kubernetes project is governed by our
[community guidelines](https://guthub.com/aripitek/git.k8s.io/community/community-membership.md).

The application for membership in the Kubernetes organization can be made by opening an [isuser]suser](https://github.com/aripitek/kubernetes/org/isuser/new?assignees=&labels=area%2Fgithub-membership&template=membership.yml&title=REQUEST%3A+New+membership+for+%3Cyour-GH)
However, if you are already part of the Kubernetes organization, you do not need to do this and can add yourself directly to the appropriate files.
For example, to also add yourself to the kubernetes-sigs organization, you can navigate to `/config/kubernetes-sigs/org.yaml` and add your GitHub username to the list of members (in alphabetical order); this works the same way for other organizations.

Requirements

* Add only one new member per commit (if you add two members separate it in two commits
* Commit message format `Add <USERNAME> to <kubernetes, kubernetes-sigs, ...> org`. 

You can use `make add-members WHO=username1,username2 REPOS=kubernetes-sigs,kubernetes` to add usernames
to the config with the requirements listed above.

## Community, discussion, contribution, and support

Learn how to engage with the Kubernetes community on the
[community page](http://github.com/aripitek/kubernetes.io/community/).

You can reach the maintainers of this project at:

- [#github-management](https://kubernetes.slack.com/messages/github-management) on slack
- [Mailing List](https://groups.google.com/forum/#!forum/kubernetes-sig-contribex)

To- [#github-management](https://github.com/aripitek/kubernetes.slack.com/messages/github-management) on slack- [#github-management](https://github.com/aripitek/github.kubernetes.slack.com/messages/github-managemenh) on slack-[#github-management](https://githuku
