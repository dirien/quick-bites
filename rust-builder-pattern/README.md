# Design Patterns in Rust 🦀: An Introduction to the Builder Pattern

## Introduction

In this blog article, I want to show talk about a design pattern implementation in `Rust 🦀`: The Builder Pattern. But before we start, let us take the time to understand what design patterns are and why we should use them in our projects.

![Design patterns with lego](https://teachyourkidscode.com/wp-content/uploads/2020/08/image4-1024x768.jpeg align="center")

### What Is a Design Pattern?

Design patterns are like pre-made blueprints that we can customize to solve recurring design problems in our code. Why do I emphasize the word "customize"? Because we can't just look up a design pattern and simply copy it into our code unlike off-the-shelf libraries and frameworks we are used to working with.

Think about design patterns as a more general concept for solving your code problem. You can follow the details of a pattern and implement the solution that fits the requirements of your code.

### Are Design Patterns the Same As Algorithms?

It happens often that developers, especially beginners, confuse design patterns with algorithms. An algorithm defines a set of actions that solves the problem, while a design pattern describes the solution on a higher level. The code of two developers implementing the same pattern can look very different.

### Should I Learn Design Patterns?

Honestly, you can work as a developer without knowing any design patterns. And you are not alone with this. So should you spend time learning design patterns?

* Design patterns are battle-proved and tested solutions for common problems in software design. Even if you don't face a problem that can be solved using a pattern it can teach how to solve problems using principles of software development.

* Design patterns will facilitate communication with other developers. You have now a common vocabulary to communicate effectively. Instead, of describing your proposal for a code changer in detail, you can just say: "We should use the Prototype pattern here!".


### Are Design Patterns Without Any Cons?

As always, where there is sunshine, there are also shadows. Some common points the criticizers of design patterns will bring up are:

* Design patterns lack the proper evidence that they work. The argument is that they are just workarounds for a problem that should have been solved in the programming language itself.

* Some argue that design patterns are a sign of weakness in the programming language. Thinking that design patterns are used as replacements for the absence of a feature in the language.

* Some developers, especially senior ones, tend to see patterns where none exist. This leads to over-engineered, bad or complicated code. *Slapping* a pattern on every problem without investigating the problem will make the code unnecessarily complex.

* If you learned more about design patterns you may try to incorporate them into your code without justifying the need for it.


Uff, that was now a bit more than I initially planned to write as an introduction. So lets us jump without further ado into the Builder Pattern and try an implementation in `Rust 🦀`.

![Ford Motor Company | History, Headquarters, & Facts | Britannica](https://cdn.britannica.com/67/94667-050-2DD1CB01/Dagenham-photo-cars-one-assembly-line-pictures-1931.jpg align="center")

## The Builder Pattern

### What Is the Builder Pattern?

The Builder Pattern belongs to the category of creational patterns. Creational patterns are all about the mechanism of creating objects by controlling their construction process. The Builder lets you construct complex objects step by step and allows you to create different representations of an object by using the same construction code.

### The Problem

Let us create a simple example to show the creation of an object without using the Builder pattern first. Our example is a `KubernetesCluster` that has the properties `name`, `version`, `auto_upgrade` and `node_pool`. The `node_pool` is an optional property that can be set to `None` if the cluster does not have any nodes or a `Vec` of `Node` objects.

We want to make the `KubernetesCluster` struct public, but we want to keep the fields private.

```rust
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
```

First, we create an implementation block for the `KubernetesCluster` struct. In this block, we are going to implement a `new` method which will create a new instance of the `KubernetesCluster` struct and takes the `name` and `version` as parameters. The `auto_upgrade` and `node_pool` fields will be set to `false` and `None` respectively.

```rust
impl KubernetesCluster {
    fn new(name: String, version: String) -> Self {
        Self {
            name,
            version,
            auto_upgrade: false,
            node_pool: None,
        }
    }
}
```

This will work for basic cases, but what if we want to activate the `auto_upgrade` feature or want to add a node pool? `Rust 🦀` does not allow function overloading, so we can't create a new `new` method for this or any further cases. Instead, we have to create another method with a different name that will allow us to set the `auto_upgrade` field.

```rust
impl KubernetesCluster {
    fn new(name: String, version: String) -> Self {
        Self {
            name,
            version,
            auto_upgrade: false,
            node_pool: None,
        }
    }
    fn new_with_auto_upgrade(name: String, version: String, auto_upgrade: bool) -> Self {
        Self {
            name,
            version,
            auto_upgrade,
            node_pool: None,
        }
    }
}
```

To go further, we will finally create a `new_complete` method that will allow us to set all fields.

```rust
impl KubernetesCluster {
    fn new(name: String, version: String) -> Self {
        Self {
            name,
            version,
            auto_upgrade: false,
            node_pool: None,
        }
    }
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
        node_pool: Option<Vec<Nodes>>,
    ) -> Self {
        Self {
            name,
            version,
            auto_upgrade,
            node_pool,
        }
    }
}
```

In our `main` function, we will see how to use the three constructors we just created. We create the variables `name` and `version` and a vector of `Nodes` and then create a very basic cluster with the `new` method. We're also going to create a cluster using the `new_with_auto_upgrade` constructor and finally a cluster with all fields set.

```rust
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

    let basic_cluster = KubernetesCluster::new(name.clone(), version.clone());

    let auto_upgrade_cluster = KubernetesCluster::new_with_auto_upgrade(
        name.clone(),
        version.clone(),
        true,
    );

    let complete_cluster = KubernetesCluster::new_complete(
        name.clone(),
        version.clone(),
        true,
        Some(nodes),
    );
}
```

### Applying the Builder Pattern

The code above is compiling but the API of the `KubernetesCluster` can be improved. Right now we have three different constructor functions with different names and the list of parameters that get passed to each function keeps growing. And if we add a new field to the `KubernetesCluster` struct, we have to add a new constructor function for it with an even longer argument list.

To prevent this multiplication of constructors, we can now apply the Builder Pattern. First, we add a new struct called `KubernetesClusterBuilder` that will have the same fields as the `KubernetesCluster` struct except that `name` and `version` are the only non-optional fields.

```rust
pub struct KubernetesClusterBuilder {
    name: String,
    version: String,
    auto_upgrade: Option<bool>,
    node_pool: Option<Vec<Node>>,
}
```

Next, we create an implementation block for the `KubernetesClusterBuilder` struct and add methods to set each of the optional fields. Each method will be named after the field it is set and will take a mutable reference to self and the value of the field as an argument. Inside the body of the method, we set the field to the value that was passed and return a mutable reference to self.

The last method of the `KubernetesClusterBuilder` struct is called `build` and will take a mutable reference to self as an argument and constructs a new `KubernetesCluster` instance. For `auto_upgrade` we use the `unwrap_or_default` method to either use the value that was past or set the default value of `false`.

The `KubernetesClusterBuilder` is now finished!

```rust
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
```

Now we head back to the `KubernetesCluster` and delete the constructors for the fields. We have now only a single constructor called `new` that takes the `name` and `version` and will return a `KubernetesClusterBuilder` instance.

`name` and `version` will be passed through the `KubernetesClusterBuilder` instance, while the other fields will be set `None` variants.

```rust
impl KubernetesCluster {
    fn new(name: String, version: String) -> KubernetesClusterBuilder {
        KubernetesClusterBuilder {
            name,
            version,
            auto_upgrade: None,
            node_pool: None,
        }
    }
}
```

Our implementation of the Builder Pattern is now completed. Let's see how we can use it in our `main` function. For the basic cluster, it stays the same as we had before, except that we return a `KubernetesClusterBuilder`. To get the `KubernetesCluster` we have to call the `build` method.

For the cluster with the activated auto upgrade, we call the now the `new` method and then the `auto_upgrade` method to set the value of `auto_upgrade` to `true`.

Finally, for the complete cluster, we switch to the `new` method and then call the `auto_upgrade` and `node_pool` methods to set the values and call the `build` method.

```rust
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
}
```

In the end we have again our the `KubernetesCluster` with a different level of configuration, but we now have only one constructor function and a bunch of methods to build up the configuration. With the help of this pattern, we can easily extend the configuration.

### Further Improvements With Using a `KubernetesDirector`

We could extract a series of calls to construct a `KubernetesCluster` into a separate struct called `KubernetesDirector`. The `KubernetesDirector` defines the order in which to execute the building steps and provides a good place to put these various construction steps to reuse across our application.

![Origin of the term boilerplate](https://qph.cf2.quoracdn.net/main-qimg-4c362d8f8b5832852852d76cd80800a8-lq align="center")

### Avoid Boilerplate Code With the `derive_builder` macro!

`Rust 🦀` has very powerful macros called `derive_builder` that can be used to create for us the builder without that we need to write all the boilerplate code manually.

To use the `derive_builder` macro, we add the dependency via the following command:

```bash
cargo add derive_builder
```

I created a new struct called `VirtualMachine` and add the `#[derive(Builder)]` attribute to it. This will generate all the necessary code at compile time.

```rust
#[derive(Builder, Debug)]
struct VirtualMachine {
    name: String,
    size: String,
    count: u32,
}
```

In our `main` function we can now call the `VirtualMachineBuilder` and set the values via the methods and then call the `build` method.

```rust
fn main() {
    // omitted previous code
    let vm = VirtualMachineBuilder::default()
        .name("my-vm".to_owned())
        .size("small".to_owned())
        .count(1)
        .build();
}
```

## Conclusion

The Builder pattern is a very useful pattern when it comes to constructing complex objects step by step. We can reuse the same construction code to create various representations of an object. And we can isolate complex construction code from our business logic following here the `Single Responsibility Principle`.

With the help of the `derive_builder` macro we have in `Rust 🦀` also, a very handy macro to prevent us from writing boring boilerplate code.

## Links

* [Builder Pattern](https://en.wikipedia.org/wiki/Builder_pattern)

* [Unofficial Rust Book](https://rust-unofficial.github.io/patterns/patterns/creational/builder.html)


Or my articles around `Rust 🦀`

%[https://blog.ediri.io/tag/rust]
