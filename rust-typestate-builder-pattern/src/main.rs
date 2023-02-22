#![allow(dead_code)]
#![allow(unused_variables)]

use std::io::Result;
use typed_builder::TypedBuilder;

#[derive(Debug, Clone)]
pub struct Node {
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


#[derive(Default, Clone)]
pub struct NoVersion;

#[derive(Default, Clone)]
pub struct Version(String);

#[derive(Default, Clone)]
pub struct NoName;

#[derive(Default, Clone)]
pub struct Name(String);

#[derive(Default, Clone)]
pub struct KubernetesClusterBuilder<V,N> {
    name: N,
    version: V,
    auto_upgrade: Option<bool>,
    node_pool: Option<Vec<Node>>,
}

impl KubernetesClusterBuilder<NoVersion,NoName> {
    pub fn new() -> Self {
        Self::default()
    }
}

impl KubernetesClusterBuilder<Version,Name> {
    pub fn build(&self) -> Result<KubernetesCluster> {
        Ok(KubernetesCluster {
            name: self.name.0.clone(),
            version: self.version.0.clone(),
            auto_upgrade: self.auto_upgrade.unwrap_or_default(),
            node_pool: self.node_pool.clone(),
        })
    }
}

impl<V,N> KubernetesClusterBuilder<V,N> {
    pub fn name(self, name: impl Into<String>) -> KubernetesClusterBuilder<V,Name> {
        let Self {
            version,
            auto_upgrade,
            node_pool,
            ..
        } = self;
        KubernetesClusterBuilder {
            name: Name(name.into()),
            version,    
            auto_upgrade,
            node_pool,
        }
    }

    pub fn version(self, version: impl Into<String>) -> KubernetesClusterBuilder<Version,N> {
        let Self {
            name,
            auto_upgrade,
            node_pool,
            ..
        } = self;
        KubernetesClusterBuilder {
            name,
            version: Version(version.into()),
            auto_upgrade,
            node_pool,
        }
    }

    pub fn auto_upgrade(mut self, auto_upgrade: bool) -> Self {
        self.auto_upgrade = Some(auto_upgrade);
        self
    }

    pub fn node_pool(mut self, node_pool: Vec<Node>) -> Self {
        self.node_pool = Some(node_pool);
        self
    }
}

#[derive(Debug, TypedBuilder)]
pub struct VirtualMachine {
    name: String,
    size: String,
    #[builder(default)]
    count: u32,
}

fn main() -> Result<()> {
    println!("Hello, world!");

    let name = "my-cluster".to_owned();
    let version = "1.25.0".to_owned();

    let nodes = vec![
        Node {
            name: "node-1".to_owned(),
            size: "small".to_owned(),
            count: 1,
        }
    ];

    let cluster_builder = KubernetesClusterBuilder::new()
        //.name(name.clone())
        .version(version.clone())
        .node_pool(nodes.clone())
        .auto_upgrade(true);

    let auto_upgrade_cluster = cluster_builder.build()?;

    println!("{auto_upgrade_cluster:#?}");

    let vm = VirtualMachine::builder()
        //.name("my-vm".to_owned())
        .size("small".to_owned())
        .count(1)
        .build();

    Ok(())
}
