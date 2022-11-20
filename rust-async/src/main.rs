use k8s_openapi::api::core::v1::Pod;
use kube::{
    api::{Api, ListParams},
    Client,
};
use kube::api::ObjectList;

#[tokio::main]
async fn main() {
    println!("Hello, world!");
    let mut handles = vec![];
    for i in 0..2 {
        let handle = tokio::spawn(async move {
            get_my_pods(i).await;
        });
        handles.push(handle);
    }
    for handle in handles {
        handle.await.unwrap();
    }
}

async fn get_my_pods(i: i32) {
    println!("[{i}] Get all my pods");
    let pods1 = get_all_pods_in_namespace("default").await;
    println!("[{i}] Got {} pods", pods1.items.len());
    let pods2 = get_all_pods_in_namespace("kube-system").await;
    println!("[{i}] Got {} pods", pods2.items.len());
}

async fn get_all_pods_in_namespace(namespace: &str) -> ObjectList<Pod> {
    println!("Get all my pods in a namespace {}", namespace);
    let client = Client::try_default().await;
    let api = Api::<Pod>::namespaced(client.unwrap(), namespace);
    let lp = ListParams::default();
    api.list(&lp).await.unwrap()
}


/*
if pods1.items.len() > 0 {
        println!("Found {} pods", pods1.items.len());
        for p in pods {
            println!("Found Pod: {}", p.metadata.name.unwrap().as_str());
        }
    } else {
        println!("No pods found");
    }
 */