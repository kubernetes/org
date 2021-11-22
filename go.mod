module k8s.io/org

go 1.16

require (
	github.com/bmatcuk/doublestar v1.3.4
	github.com/ghodss/yaml v1.0.0
	github.com/go-git/go-git/v5 v5.5.2
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	k8s.io/apimachinery v0.20.2
	k8s.io/test-infra v0.0.0-20230726003218-c95b43963de2
	sigs.k8s.io/yaml v1.2.0
)

// Pin all k8s.io staging repositories to kubernetes v0.18.6
// When bumping Kubernetes dependencies, you should update each of these lines
// to point to the same kubernetes v0.KubernetesMinor.KubernetesPatch version
// before running update-deps.sh.
replace (
	cloud.google.com/go/pubsub => cloud.google.com/go/pubsub v1.3.1
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	golang.org/x/lint => golang.org/x/lint v0.0.0-20190409202823-959b441ac422
	gopkg.in/yaml.v3 => gopkg.in/yaml.v3 v3.0.0-20190709130402-674ba3eaed22
	k8s.io/client-go => k8s.io/client-go v0.20.2
)
