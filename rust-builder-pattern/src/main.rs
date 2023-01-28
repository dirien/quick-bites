#[macro_use]
extern crate derive_builder;

#[derive(Debug, Clone)]
struct Node {
    name: String,
    size: String,
    count: u32,
}

#[derive(Debug)]
pub struct KubernetesCluster {
    name: String,
    version: String,
    auto_upgrade: bool,
    node_pool: Option<Vec<Node>>,
}

impl KubernetesCluster {
    fn new(name: String, version: String) -> KubernetesClusterBuilder {
        KubernetesClusterBuilder {
            name,
            version,
            auto_upgrade: None,
            node_pool: None,
        }
    }

    /*
    // remove this after you have implemented the builder pattern
    fn new_with_auto_upgrade(name: String, version: String, auto_upgrade: bool) -> Self {
        Self {
            name,
            version,
            auto_upgrade,
            node_pool: None,
        }
    }
    fn new_complete(
        name: String,
        version: String,
        auto_upgrade: bool,
        node_pool: Option<Vec<Node>>,
    ) -> Self {
        Self {
            name,
            version,
            auto_upgrade,
            node_pool,
        }
    }
    */
}


pub struct KubernetesClusterBuilder {
    name: String,
    version: String,
    auto_upgrade: Option<bool>,
    node_pool: Option<Vec<Node>>,
}

impl KubernetesClusterBuilder {
    fn auto_upgrade(&mut self, auto_upgrade: bool) -> &mut Self {
        self.auto_upgrade = Some(auto_upgrade);
        self
    }

    fn node_pool(&mut self, node_pool: Vec<Node>) -> &mut Self {
        self.node_pool = Some(node_pool);
        self
    }

    fn build(&mut self) -> KubernetesCluster {
        KubernetesCluster {
            name: self.name.clone(),
            version: self.version.clone(),
            auto_upgrade: self.auto_upgrade.unwrap_or_default(),
            node_pool: self.node_pool.clone(),
        }
    }
}

#[derive(Builder, Debug)]
struct VirtualMachine {
    name: String,
    size: String,
    count: u32,
}


fn main() {
    let name = "my-cluster".to_owned();
    let version = "1.25.0".to_owned();

    let nodes = vec![
        Node {
            name: "node-1".to_owned(),
            size: "small".to_owned(),
            count: 1,
        }
    ];

    let basic_cluster = KubernetesCluster::new(name.clone(), version.clone()).build();

    let auto_upgrade_cluster = KubernetesCluster::new(
        name.clone(),
        version.clone(),
    ).auto_upgrade(true)
        .build();

    let complete_cluster = KubernetesCluster::new(
        name.clone(),
        version.clone(),
    ).auto_upgrade(true)
        .node_pool(nodes)
        .build();

    let vm = VirtualMachineBuilder::default()
        .name("my-vm".to_owned())
        .size("small".to_owned())
        .count(1)
        .build();
}
