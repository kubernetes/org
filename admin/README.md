# Apply kubernetes configuration

Merge a PR that changes some `config/foo/org.yaml` and then run the following:
```shell
# Displays what it would do without making changes until you add the confirm flag
bazel run //admin:update -- --github-token-path ~/path-to-my-token # --confirm
```

This will default to a dry-run mode, displaying what changes it intends to make without actually updating anything on github.
It will apply the change if you send it the `--confirm` flag.

It also runs the `bazel test //config:all` unit tests to sanity check the config.

Assuming everything works the tool should output something like the following:
```console
{"client":"github","component":"peribolos","level":"info","msg":"Throttle(300, 100)","time":"2018-08-10T17:42:15-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"GetOrg(kubernetes-incubator)","time":"2018-08-10T17:42:15-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgInvitations(kubernetes-incubator)","time":"2018-08-10T17:42:17-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-incubator, admin)","time":"2018-08-10T17:42:17-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-incubator, member)","time":"2018-08-10T17:42:17-07:00"}
{"component":"peribolos","level":"info","msg":"Skipping team and team member configuration","time":"2018-08-10T17:42:17-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"GetOrg(kubernetes-retired)","time":"2018-08-10T17:42:17-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgInvitations(kubernetes-retired)","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-retired, admin)","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-retired, member)","time":"2018-08-10T17:42:18-07:00"}
{"component":"peribolos","level":"info","msg":"Waiting for calebamiles to accept invitation to kubernetes-retired","time":"2018-08-10T17:42:18-07:00"}
{"component":"peribolos","level":"info","msg":"Skipping team and team member configuration","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"GetOrg(kubernetes-sigs)","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"EditOrg(kubernetes-sigs, {steering-private@kubernetes.io    Kubernetes SIGs Org for Kubernetes SIG-related work true true read false})","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgInvitations(kubernetes-sigs)","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-sigs, admin)","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-sigs, member)","time":"2018-08-10T17:42:18-07:00"}
{"component":"peribolos","level":"info","msg":"Waiting for calebamiles to accept invitation to kubernetes-sigs","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes-sigs, carolynvs)","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes-sigs, jeremyrickard)","time":"2018-08-10T17:42:18-07:00"}
{"component":"peribolos","level":"info","msg":"Skipping team and team member configuration","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"GetOrg(kubernetes)","time":"2018-08-10T17:42:18-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgInvitations(kubernetes)","time":"2018-08-10T17:42:19-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes, admin)","time":"2018-08-10T17:42:19-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes, member)","time":"2018-08-10T17:42:19-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes, ianychoi)","time":"2018-08-10T17:42:21-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes, akutz)","time":"2018-08-10T17:42:21-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes, gochist)","time":"2018-08-10T17:42:21-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes, jeremyrickard)","time":"2018-08-10T17:42:21-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes, fanzhangio)","time":"2018-08-10T17:42:21-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes, dvonthenen)","time":"2018-08-10T17:42:21-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes, rosti)","time":"2018-08-10T17:42:21-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"RemoveOrgMembership(kubernetes, bart0sh)","time":"2018-08-10T17:42:21-07:00"}
{"component":"peribolos","level":"info","msg":"Skipping team and team member configuration","time":"2018-08-10T17:42:21-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"GetOrg(kubernetes-client)","time":"2018-08-10T17:42:21-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgInvitations(kubernetes-client)","time":"2018-08-10T17:42:22-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-client, admin)","time":"2018-08-10T17:42:22-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-client, member)","time":"2018-08-10T17:42:22-07:00"}
{"component":"peribolos","level":"info","msg":"Waiting for calebamiles to accept invitation to kubernetes-client","time":"2018-08-10T17:42:22-07:00"}
{"component":"peribolos","level":"info","msg":"Skipping team and team member configuration","time":"2018-08-10T17:42:22-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"GetOrg(kubernetes-csi)","time":"2018-08-10T17:42:22-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgInvitations(kubernetes-csi)","time":"2018-08-10T17:42:22-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-csi, admin)","time":"2018-08-10T17:42:22-07:00"}
{"client":"github","component":"peribolos","level":"info","msg":"ListOrgMembers(kubernetes-csi, member)","time":"2018-08-10T17:42:23-07:00"}
{"component":"peribolos","level":"info","msg":"Waiting for calebamiles to accept invitation to kubernetes-csi","time":"2018-08-10T17:42:23-07:00"}
{"component":"peribolos","level":"info","msg":"Skipping team and team member configuration","time":"2018-08-10T17:42:23-07:00"}
```

Happy administering!
