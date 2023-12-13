package hooks

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path"
	"slices"
	"time"

	"github.com/cucumber/godog"
	messages "github.com/cucumber/messages/go/v21"
	cp "github.com/otiai10/copy"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"

	"github.com/filariow/mctest/demo/e2e/internal/assets"
	einfra "github.com/filariow/mctest/demo/e2e/internal/infra"
	"github.com/filariow/mctest/demo/e2e/internal/scheme"
	econtext "github.com/filariow/mctest/pkg/context"
	"github.com/filariow/mctest/pkg/infra"
	"github.com/filariow/mctest/pkg/infra/clusterapi"
	"github.com/filariow/mctest/pkg/kube"
	"github.com/filariow/mctest/pkg/testrun"
)

func setTimeout(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
	tctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	return testrun.TimeoutContextCancelIntoContext(tctx, cancel), nil
}

func injectProvisioners(ctx context.Context, s *godog.Scenario) (context.Context, error) {
	k, err := einfra.ClusterFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	hmm, err := k.ParseResources(ctx, assets.DefaultClusterSpec)
	if err != nil {
		return ctx, err
	}

	ns, err := einfra.AuxiliaryScenarioNamespaceFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	scopedHostManifests := make([]unstructured.Unstructured, len(hmm))
	for i, m := range hmm {
		m.SetNamespace(*ns)
		scopedHostManifests[i] = m
	}

	hostProvisioners := map[string]infra.ClusterProvisioner{}
	dp := clusterapi.NewClusterAPIProvisioner(k.Kubernetes, scopedHostManifests, s.Id)
	if n := dp.NumClustersProvisionedInProvisionRound(); n != 1 {
		panic(fmt.Sprintf("host provider is expected to provision just 1 cluster, found %d", n))
	}

	hostProvisioners["default"] = dp
	ctx = einfra.ProvisionersIntoContext(ctx, hostProvisioners)

	return ctx, nil
}

// injects the configured cluster into context
func injectManagementCluster(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	// build cluster clients
	cfg, err := kube.GetRESTConfig()
	if err != nil {
		return ctx, err
	}

	k, err := kube.New(cfg, scheme.DefaultSchemeHost, true)
	if err != nil {
		return ctx, err
	}

	h := infra.NewCluster(k)

	// inject management cluster
	ctx = einfra.ManagementClusterIntoContext(ctx, *h)
	ctx = einfra.ClusterIntoContext(ctx, *h)
	return ctx, nil
}

// if scenario's tag contains @dedicated-cluster, this hook will provision a dedicated cluster
func injectDedicatedClusterIfRequired(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	// dedicated host not requested, inject management cluster as the host cluster
	if !isDedicatedClusterRequired(sc) {
		log.Printf("dedicated cluster required, provisioning dedicated cluster")
		ctx, err := provisionAndInjectDedicatedCluster(ctx, sc)
		if err != nil {
			return ctx, err
		}

		return prepareScenarioNamespaces(ctx, sc)
	}

	log.Printf("dedicated cluster not required")
	return ctx, nil

}

func provisionAndInjectDedicatedCluster(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	// TODO: is this needed for this demo?

	// retrieve provisioner
	hostProvisioners, err := einfra.ProvisionersFromContext(ctx)
	if err != nil {
		return ctx, econtext.ErrKeyNotFound
	}

	p, ok := (*hostProvisioners)[defaultClusterProvisioner]
	if !ok {
		return ctx, fmt.Errorf(
			"host provisioner %s for scenario %s not found in registered ones: %v",
			defaultClusterProvisioner, sc.Name, *hostProvisioners)
	}

	// provision host cluster
	if err := p.Provision(ctx); err != nil {
		return ctx, err
	}

	if err := p.WaitForProvisionedClusters(ctx); err != nil {
		return ctx, err
	}

	// retrieve admin kubeconfig
	cfgs, err := p.GetAllAdminKubeconfigs(ctx)
	if err != nil {
		return ctx, err
	}
	cfg := func() *rest.Config {
		for _, v := range cfgs {
			return &v
		}

		panic("expected at least one kubeconfig from cluster provisioner, got none")
	}()

	// build kube.Kubernetes
	k, err := kube.New(cfg, scheme.DefaultSchemeHost, true)
	if err != nil {
		return ctx, err
	}

	// TODO: test connection to the cluster
	// TODO: retry fetching config and building kube client if no connection available

	// inject into context
	h := infra.NewCluster(k)
	return einfra.ClusterIntoContext(ctx, *h), err
}

func isDedicatedClusterRequired(s *godog.Scenario) bool {
	return slices.ContainsFunc(
		s.Tags,
		func(e *messages.PickleTag) bool {
			return e.Name == TagDedicatedCluster
		})
}

func prepareScenarioNamespaces(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	// get host cluster from context
	h, err := einfra.ClusterFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	// create the scenario namespace
	an, err := createAuxiliaryTestNamespace(ctx, h, sc.Id)
	if err != nil {
		return ctx, err
	}
	n, err := createTestNamespace(ctx, h, sc.Id)
	if err != nil {
		return ctx, err
	}

	// inject scenario namespace in context
	ctx = einfra.AuxiliaryScenarioNamespaceIntoContext(ctx, an.Name)
	ctx = einfra.ScenarioNamespaceIntoContext(ctx, n.Name)

	// if the host is dedicated for this scenario, provide the admin client
	if h.IsDedicated {
		return ctx, nil
	}

	// otherwise build a namespaced client
	nk, err := prepareNamespacedClient(ctx, h.Kubernetes, n.Name)
	if err != nil {
		return ctx, err
	}

	// update host cluster into context
	nh := infra.NewCluster(nk)
	return einfra.ClusterIntoContext(ctx, *nh), nil
}

func createTestNamespace(ctx context.Context, cluster *infra.Cluster, scenarioId string) (*corev1.Namespace, error) {
	return createNamespace(ctx, cluster, "test", scenarioId)
}

func createAuxiliaryTestNamespace(ctx context.Context, cluster *infra.Cluster, scenarioId string) (*corev1.Namespace, error) {
	return createNamespace(ctx, cluster, "test-aux", scenarioId)
}

func createNamespace(ctx context.Context, cluster *infra.Cluster, prefix, scenarioId string) (*corev1.Namespace, error) {
	n := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", prefix, scenarioId),
			Labels: map[string]string{
				"scope":    "test",
				"scenario": scenarioId,
			},
		},
	}
	return cluster.Cli.CoreV1().Namespaces().Create(ctx, &n, metav1.CreateOptions{})
}

func prepareNamespacedClient(ctx context.Context, k *kube.Kubernetes, ns string) (*kube.Kubernetes, error) {
	var err error

	// create service account
	sa := &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "test-runner"}}
	sa, err = k.Cli.CoreV1().ServiceAccounts(ns).Create(ctx, sa, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// create token
	r := &authenticationv1.TokenRequest{Spec: authenticationv1.TokenRequestSpec{}}
	if d, ok := ctx.Deadline(); ok {
		es := int64(d.Sub(time.Now()).Seconds())
		r.Spec.ExpirationSeconds = &es
	}
	r, err = k.Cli.CoreV1().ServiceAccounts(ns).CreateToken(ctx, sa.Name, r, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// create admin role
	ro := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: sa.Name},
		Rules: []rbacv1.PolicyRule{
			{
				Verbs:     []string{"*"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
				// ResourceNames: []string{},
			},
		},
	}
	ro, err = k.Cli.RbacV1().Roles(ns).Create(ctx, ro, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// create rolebinding
	rb := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: sa.Name},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				APIGroup:  "",
				Name:      sa.Name,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     ro.Name,
		},
	}
	rb, err = k.Cli.RbacV1().RoleBindings(ns).Create(ctx, rb, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	// bake client
	cfg := *k.Cfg
	cfg.Impersonate = rest.ImpersonationConfig{
		UserName: fmt.Sprintf("system:serviceaccount:%s:%s", ns, sa.Name),
		UID:      string(sa.GetUID()),
	}
	return kube.New(&cfg, scheme.DefaultSchemeHost, false)
}

func hookPrepareScenarioTestFolder(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	// prepare folder
	tf := path.Join("runs", sc.Id)
	if itf, err := os.Stat(tf); err != nil {
		if !os.IsNotExist(err) {
			return ctx, err
		}
	} else {
		if !itf.IsDir() {
			return ctx, fmt.Errorf("expected %s to be a temporary folder, found a file", tf)
		}
		if err := os.RemoveAll(tf); err != nil {
			return ctx, err
		}
	}

	if err := os.MkdirAll(tf, 0755); err != nil {
		return ctx, err
	}

	opts := cp.Options{
		AddPermission: 0600,
		OnDirExists: func(src, dest string) cp.DirExistsAction {
			return cp.Replace
		},
		PreserveOwner: true,
	}
	if err := cp.Copy("base", tf, opts); err != nil {
		return ctx, err
	}

	return testrun.TestFolderIntoContext(ctx, tf), nil
}
