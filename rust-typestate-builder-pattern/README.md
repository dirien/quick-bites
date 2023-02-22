# Design Patterns in Rust 🦀: Upgrading the Builder Pattern using Typestate
## Introduction

In this blog article, I want to upgrade my builder pattern implementation in `Rust 🦀` using the `Typestate` pattern. If you are not familiar with the builder pattern or design patterns in general, I highly recommend you to read my previous blog article:

https://blog.ediri.io/design-patterns-in-rust-an-introduction-to-the-builder-pattern

### What is the Typestate pattern?

The `Typestate` pattern encodes the runtime state of an object into its compile-time type. This allows us to move certain types of errors from runtime to compile-time, which is a great way to improve the quality of our code as the developer gets immediate feedback. This feedback can be shown also in the IDE and is therefore much more convenient as you can avoid operations that are not allowed in the current state.

Programs using the `Typestate` pattern will have the following characteristics:

* Operations that change the type-level state of an object in addition/instead of changing the runtime state so that operations that are not allowed in the current state are not possible anymore.

* A fixed way of encoding the state of an object into its type so that any attempt to change the state in a way that is not allowed will result in a compile-time error.

* Certain operations on an object are only allowed in certain states. If an operation is not allowed in the current state, the compiler will not allow it.


In short, we can say that the `Typestate` pattern parts of the dynamic information in the compiler space and have a way to check ahead of time if an operation is allowed in the current state.

## Revisit the Builder Pattern

This is the code from our last blog article when we created the builder pattern in `Rust 🦀`:

```rust
#![allow(dead_code)]
#![allow(unused_variables)]

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
pub struct KubernetesClusterBuilder {
    name: Option<String>,
    version: Option<String>,
    auto_upgrade: Option<bool>,
    node_pool: Option<Vec<Node>>,
}

impl KubernetesClusterBuilder {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn name(&mut self, name: String) -> &mut Self {
        self.name = Some(name);
        self
    }

    pub fn version(&mut self, version: String) -> &mut Self {
        self.version = Some(version);
        self
    }

    pub fn auto_upgrade(&mut self, auto_upgrade: bool) -> &mut Self {
        self.auto_upgrade = Some(auto_upgrade);
        self
    }

    pub fn node_pool(&mut self, node_pool: Vec<Node>) -> &mut Self {
        self.node_pool = Some(node_pool);
        self
    }

    fn build(&mut self) -> KubernetesCluster {
        KubernetesCluster {
            name: self.name.clone().unwrap(),
            version: self.version.clone().unwrap_or_default(),
            auto_upgrade: self.auto_upgrade.unwrap_or_default(),
            node_pool: self.node_pool.clone(),
        }
    }
}

fn main() {
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

    let mut cluster_builder = KubernetesClusterBuilder::new();
    cluster_builder.name(name.clone())
        .version(version.clone())
        .auto_upgrade(true);

    let auto_upgrade_cluster = cluster_builder.build();

    println!("{:#?}", auto_upgrade_cluster);
}
```

In our scenario, we want to make sure that `name` and `version` are always called before `build` can be called.

We are going to redefine the `KubernetesClusterBuilder` struct to be generic over `V` and `N`

```rust
#[derive(Default, Clone)]
pub struct KubernetesClusterBuilder<V, N> {
    name: N,
    version: V,
    auto_upgrade: Option<bool>,
    node_pool: Option<Vec<Node>>,
}
```

The generics `V` and `N` will be used to encode the state of the builder. The fields `name` and `version` can be either set as `String` or not set at all.

```rust
#[derive(Default, Clone)]
pub struct NoVersion;

#[derive(Default, Clone)]
pub struct Version(String);

#[derive(Default, Clone)]
pub struct NoName;

#[derive(Default, Clone)]
pub struct Name(String);
```

When `KubernetesClusterBuilder:new` is called, neither `name` nor `version` are set. Therefore, we will use the value of the type `KubernetesClusterBuilder<NoVersion,NoName>` as the return type. In our case, we will use the `Default` trait to set the default values for the fields.

```rust
impl KubernetesClusterBuilder<NoVersion, NoName> {
    pub fn new() -> Self {
        Self::default()
    }
}
```

When `.name()` is called, we will set the value of `name` to `Name` and return the type `KubernetesClusterBuilder<V,Name>`.

```rust
  pub fn name(self, name: impl Into<String>) -> KubernetesClusterBuilder<V, Name> {
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
```

Symmetrically, when `.version()` is called, we will set the value of `version` to `Version` and return the type \`KubernetesClusterBuilder&lt;Version,N&gt;\`

```rust
pub fn version(self, version: impl Into<String>) -> KubernetesClusterBuilder<Version, N> {
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
```

The remaining methods are unchanged.

```rust
pub fn auto_upgrade(mut self, auto_upgrade: bool) -> Self {
    self.auto_upgrade = Some(auto_upgrade);
    self
}

pub fn node_pool(mut self, node_pool: Vec<Node>) -> Self {
    self.node_pool = Some(node_pool);
    self
}
```

The last method is the `build` method. This method will be called only if both `name` and `version` are set. We create a new implementation for `KubernetesClusterBuilder<Version,Name>`.

```rust
impl KubernetesClusterBuilder<Version, Name> {
    pub fn build(&self) -> Result<KubernetesCluster> {
        Ok(KubernetesCluster {
            name: self.name.0.clone(),
            version: self.version.0.clone(),
            auto_upgrade: self.auto_upgrade.unwrap_or_default(),
            node_pool: self.node_pool.clone(),
        })
    }
}
```

As the structs are tuples, we can access the value of the fields using the `0` index.

```rust
name: self .name.0.clone(),
version: self .version.0.clone(),
```

And that's it! We have created a typestate builder pattern in `Rust 🦀`. We can now try the following code.

```rust
fn main() -> Result<()> {
    // remove for brevity

    let cluster_builder = KubernetesClusterBuilder::new()
        // name is not set    
        .version(version.clone())
        .node_pool(nodes.clone())
        .auto_upgrade(true);

    let auto_upgrade_cluster = cluster_builder.build()?;

    println!("{auto_upgrade_cluster:#?}");
    Ok(())
}
```

The code will not compile as `name` is not set.

```bash
cargo run -q
error[E0599]: no method named `build` found for struct `KubernetesClusterBuilder<Version, NoName>` in the current scope
   --> src/main.rs:120:48
    |
35  | pub struct KubernetesClusterBuilder<V,N> {
    | ---------------------------------------- method `build` not found for this struct
...
120 |     let auto_upgrade_cluster = cluster_builder.build()?;
    |                                                ^^^^^ method not found in `KubernetesClusterBuilder<Version, NoName>`
    |
    = note: the method was found for
            - `KubernetesClusterBuilder<Version, Name>`
```

You should see the error also in your IDE.

![](https://cdn.hashnode.com/res/hashnode/image/upload/v1677080655270/cab570bd-5476-49c2-abad-1fc03d523b61.png align="center")

We can now try to set the name and version.

```rust
fn main() -> Result<()> {
    // remove for brevity

    let cluster_builder = KubernetesClusterBuilder::new()
        .name(name.clone())
        .version(version.clone())
        .node_pool(nodes.clone())
        .auto_upgrade(true);

    let auto_upgrade_cluster = cluster_builder.build()?;

    println!("{auto_upgrade_cluster:#?}");
    Ok(())
}
```

The code will compile fina and run without any problems.

```bash
 cargo run -q
Hello, world!
KubernetesCluster {
    name: "my-cluster",
    version: "1.25.0",
    auto_upgrade: true,
    node_pool: Some(
        [
            Node {
                name: "node-1",
                size: "small",
                count: 1,
            },
        ],
    ),
}
```

### Avoid Boilerplate Code With the `typed-builder` macro!

`Rust 🦀` has very powerful macros called `typed-builder` that can be used to create for us the `Typestate` builder without that we need to write all the boilerplate code manually.

To use the `derive_builder` macro, we add the dependency via the following command:

```bash
cargo add typed-builder
```

I reuse the struct `VirtualMachine` from the last blog article and add the `#[derive(TypedBuilder)]` attribute to it. This will generate all the necessary code at compile time.

```rust
#[derive(Debug, TypedBuilder)]
struct VirtualMachine {
    name: String,
    size: String,
    #[builder(default = 1)]
    count: u32,
}
```

In our `main` function we can now call the `VirtualMachine::builder()` and set the values via the methods and then call the `build` method but not set the `name` field.

```rust
fn main() {
    // omitted previous code
    let vm = VirtualMachine::builder()
        //.name("my-vm".to_owned())
        .size("small".to_owned())
        .count(1)
        .build();
}
```

This will not compile, and you should see the following error in your IDE.

![](https://cdn.hashnode.com/res/hashnode/image/upload/v1677080713311/a9d8bfcc-5abf-4d7a-b448-5800dd6b8c72.png align="center")

## Conclusion

Adding the `Typestate` pattern to your builder can help you to avoid errors at compile time and not at runtime. We can express the state and the allowed transitions in the type system.

But as always, there are trade-offs. The usage of generics increases the binary size and the compilation time. Most of the time the usage of `generic` still outweighs those disadvantages but there is a point where the returns are not worth overusing generics.

And of course, designing a builder with too many states would be overkill and could even have counterproductive side effects.

Finally, we also so the usage of the `typed-builder` macro that can be used to generate the builder for us without the need to write all the boilerplate code.

## Links

* [The Typestate Pattern in Rust](http://cliffle.com/blog/rust-typestate/)

* [Builder with typestate in Rust](https://www.greyblake.com/blog/builder-with-typestate-in-rust/)

* [typed-builder](https://crates.io/crates/typed-builder)


Or my articles around `Rust 🦀`

%[https://blog.ediri.io/tag/rust]
