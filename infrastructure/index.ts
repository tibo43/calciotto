import * as pulumi from "@pulumi/pulumi";
import * as scaleway from "@pulumiverse/scaleway";
import * as _null from "@pulumi/null";


const pn = new scaleway.network.PrivateNetwork("pn", {
    name: "calciotto",
    tags: [
        "development",
        "calciotto"
    ]
});
const cluster = new scaleway.kubernetes.Cluster("cluster", {
    name: "calciotto",
    version: "1.34.1",
    cni: "cilium",
    privateNetworkId: pn.id,
    deleteAdditionalResources: false,
    tags: [
        "development",
        "calciotto"
    ]
});
const pool = new scaleway.kubernetes.Pool("pool", {
    clusterId: cluster.id,
    name: "calciotto",
    nodeType: "STARDUST1-S",
    size: 0,
    minSize: 0,
    maxSize: 3,
    tags: [
        "development",
        "calciotto"
    ]
});
const kubeconfig = new _null.Resource("kubeconfig", {triggers: {
    host: cluster.kubeconfigs.apply(kubeconfigs => kubeconfigs[0].host),
    token: cluster.kubeconfigs.apply(kubeconfigs => kubeconfigs[0].token),
    cluster_ca_certificate: cluster.kubeconfigs.apply(kubeconfigs => kubeconfigs[0].clusterCaCertificate),
}}, {
    dependsOn: [pool],
});