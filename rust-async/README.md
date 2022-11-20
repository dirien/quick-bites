# How to async/await in Rust: An Introduction

## TL;DR Le code

As usual, the code is available on GitHub:


## Introduction

In this blog post, we will explore how to use the `async/await` syntax in `Rust` 🦀. We will create a simple example and then gradually add more complexity. At the end we will add the `Tokio` runtime and see how to use it.

## Prerequisites

Before we start, we need to make sure we have the following tools installed:

- [Rust](https://www.rust-lang.org)
- An IDE or text editor of your choice
- A Kubernetes cluster, I use Docker Desktop for Mac. You can use whatever you prefer.

The Kubernetes part is just for the demo, you don't need it to follow along about the `async/await` syntax.

## Initialize the project

```shell
cargo init
```

We will add the `kube` and `k8s-openapi` crates to our demo project. The usage of these libraries are not in the focus of
this article. In the future, I will write an article on how to use `Rust` 🦀 to interact with Kubernetes.

For now, just let us add the crates to our project using the following commands:

```shell
cargo add kube --features default, derive
cargo add k8s-openapi --features v1_24
```

## Async/await in Rust

The `async/await` syntax is a way to write asynchronous code in `Rust` 🦀. To demonstrate how it works, we will start with
following code:

```rust
fn main() {
    println!("Hello, world!");
}

fn get_my_pods() {
    println!("Get all my pods in default namespace");
}
```

Let's make the `get_my_pods` function asynchronous. We will use the `async` keyword to do that:

```rust
fn main() {
    println!("Hello, world!");
}

async fn get_my_pods() {
    println!("Get all my pods in default namespace");
}
```

In `Rust` 🦀 `async/wait` is a special syntax that allows us to write functions, closures, and blocks that can be paused and
yield control back to the caller. This allows other parts of the program to run while the async function is waiting and
pick up where it left off when it is ready to continue.

One advantage of using `async/await` syntax is that allows us to write code which looks like synchronous code.

The `async/await` syntax is similar to the on in other languages like JavaScript or C# with some key differences.

The `async` keyword is actually syntactic sugar for a function similar to this:

```rust
fn main() {
    println!("Hello, world!");
}

fn get_my_pods() -> impl Future<Output=()> {
    println!("Get all my pods in default namespace");
}

//async fn get_my_pods() {
//    println!("Get all my pods in default namespace");
//}
```

Async functions are a special kind of function that return a value that implements the `Future` trait. The `Future`
output type is the type that is returned by the function after it has completed. In our case, we are not returning
anything, so we use the `()` type.

A simplified version of the `Future` trait looks like this:

```rust
trait Future {
    type Output;
    fn poll(&mut self, wake: fn()) -> Poll<Self::Output>;
}

enum Poll<T> {
    Ready(T),
    Pending,
}

fn main() {
    println!("Hello, world!");
}

fn get_my_pods() -> impl Future<Output=()> {
    println!("Get all my pods in default namespace");
}
```

A `Future` is a simple state machine which can be polled to check if it is ready or not. The `poll` method returns an
enumeration with two possible values: `Ready` or `Pending`. It also accepts a callback function called `wake`.

If calling th `poll` method returns pending then the Future will continue making progress in the background until it is
ready to get polled again. The `wake` callback function is used to notify the executor that the Future is ready to be
polled again.

In Javascript this is similar to Promises, expect in `Rust` 🦀 futures are lazy and don't start executing unless they are
driven to completion by being polled.

Futures could be driven to completion by either awaiting them or giving them to an executor.

Let's see how we can await a future in `Rust` 🦀. We will add another function called `get_all_pods_in_namespace` which will
print the name of all the pods in a given namespace:

```rust
async fn get_all_pods_in_namespace(namespace: &str) -> ObjectList<Pod> {
    println!("Get all my pods in a namespace {}", namespace);
    let client = Client::try_default().await;
    let api = Api::<Pod>::namespaced(client.unwrap(), namespace);
    let lp = ListParams::default();
    api.list(&lp).await.unwrap()
}
```

In our function `get_my_pods` we will call the `get_all_pods_in_namespace` function for two namespaces and print the
number of pods in each namespace. We will use the `await` keyword to wait for the future to complete.

```rust
async fn get_my_pods() {
    println!("Get all my pods");
    let pods1 = get_all_pods_in_namespace("default").await;
    println!("Got {} pods", pods1.items.len());
    let pods2 = get_all_pods_in_namespace("kube-system").await;
    println!("Got {} pods", pods2.items.len());
}
```

This is what allows us to write asynchronous code that looks like synchronous code. The `await` keyword will also pause
the execution of the current function yielding control back to the runtime.

To understand how this works, imagine the process of calling get_my_pods as a state machine being an enum with three
states:

```rust
enum FutureStateMachine {
    State1,
    State2,
    State3,
}
```

When `get_my_pods` is first called, all the code in the function is executed until the first `await` point. The
future will return the `Pending` state because it's waiting for the Kubernetes API to return the list of pods. Once the
list pods is returned, `get_my_pods` will notify its executor that it is ready to be polled again. The executor will
then resume the execution of the function and continue until the next `await` point. This process will continue until
the second `await` point is reached. The code in the state 2 will be executed synchronously everything up to the
second `await` point. And again the function will return the `Pending` state because it's waiting for the Kubernetes API
to return. Once the list pods is returned, `get_my_pods` will notify its executor that it is ready to be polled again.
On the third state the rest of the code will be executed synchronously and the function will return the `Ready` state as
we are at the end of the function.

Now we know how the `async/await` syntax works, now let's try to call the `get_my_pods` function from the `main`
function.

If we try to call the `get_my_pods` function from the `main` function, we will get a compiler error:

```rust
fn main() {
    println!("Hello, world!");
    get_my_pods();
}
```

You will get a warning which states that futures do nothing unless you `.await` or poll them. So let's try to await the
function:

```rust
fn main() {
    println!("Hello, world!");
    get_my_pods().await;
}
```

Now we get a compiler error which states that `await` is only allowed in `async` functions. So let's make the `main`
function `async`:

```rust
async fn main() {
    println!("Hello, world!");
    get_my_pods().await;
}
```

Now we get a compiler error which states that `main` is not allowed to be `async`.

So how do we call an `async` function from the `main` function? Futures can be driven to completion in two ways:

1. By awaiting them.
2. Or manually polling them until they are ready.

On futures inside other futures, we can use the `await` keyword. But if we want to call an `async` function from the top
most level of our program, we need to manually poll the future until it is ready. That code is called a `runtime` or
an `executor`.

A `runtime` is responsible for polling the top level futures until they are ready. It is also responsible for running
multiple futures in parallel. The standard library does not provide a runtime, but there are many community built
runtimes available. We will use the most popular one called `tokio` in this tutorial.

## What is Tokio?

`Tokio` is an asynchronous runtime for the `Rust` 🦀 programming language. It provides the building blocks needed for writing
network applications. It gives the flexibility to target a wide range of systems, from large servers with dozens of
cores to small embedded devices.

At a high level, `Tokio` provides a few major components:

- A multi-threaded runtime for executing asynchronous code.
- An asynchronous version of the standard library.
- A large ecosystem of libraries.

The advantage of using `Tokio` is that it is fast, reliable, easy to use and very flexible.

To add `Tokio` to our project, we need run the following command:

```bash
cargo add tokio --features full
```

This will add the `tokio` crate to our project and enable all the features.

Now we can use the attribute `tokio::main` macro to run our `main` function:

```rust
#[tokio::main]
async fn main() {
    println!("Hello, world!");
    get_my_pods().await;
}
```

This specifies that the `main` function is an asynchronous function and will be executed by the `Tokio` runtime.

Let's run our program:

```bash
cargo run
```

You should see the following output:

```bash
Hello, world!
Get all my pods
Get all my pods in a namespace default
Got 0 pods
Get all my pods in a namespace kube-system
Got 9 pods
```

You can see that our print statements are printed in the order they are called. As mentioned before: Futures are lazy
and do nothing unless you await them or poll them.

We can save the result of the `get_my_pods` function in a variable and call `await` on it later on:

```rust
#[tokio::main]
async fn main() {
    println!("Hello, world!");
    let p = get_my_pods();
    println!("Where are my pods?");
    p.await;
}
```

Let's run our program again:

```bash
cargo run
```

You should see the following output:

```bash
Hello, world!
Where are my pods?
Get all my pods
Get all my pods in a namespace default
Got 0 pods
Get all my pods in a namespace kube-system
Got 9 pods
```

This time the println statement `Where are my pods?` is printed before the `get_my_pods` function is called.

On benefit of futures being lazy is that they are a zero cost abstraction. This means you won't incur a runtime cost
unless you actually await the future. Another benefit is that you can cancel a future at any time. To cancel a future
all you need to do is to stop polling it.

Currently, we are not taking advantage of the asynchronous nature of the `get_my_pods` function because everything is
running serially. To make our run concurrently we can use a tokio task. A task is a lightweight non-blocking unit of
execution.

Let's change the code to use a task. First we create an empty vector to store our task handles, and then we will create a
loop with two iterations. In each iteration we will create a task and store the handle in the vector.

We are passing an `async` block to the `spawn` function. We can use the `move` keyword with the `async` block so that
the `async` block can capture the variables from the outer scope. In this case we are capturing the `i` variable and
pass it to the `get_my_pods` function.

At the end of the main we loop through the vector of task handles and call `await` on each of them.

```rust
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
```

So let's run our program again:

```bash
cargo run
```

You should see the following output:

```bash
Hello, world!
[0] Get all my pods
Get all my pods in a namespace default
[1] Get all my pods
Get all my pods in a namespace default
[0] Got 0 pods
Get all my pods in a namespace kube-system
[1] Got 0 pods
Get all my pods in a namespace kube-system
[0] Got 9 pods
[1] Got 9 pods
```

You can see that the two tasks are running concurrently. By default, `Tokio` uses a thread pool to execute tasks. This
allows tasks to execute tasks in parallel.

There is also the possibility to force tokio to run tasks on the same thread. This can be done by using
the `current_thread` flavor of the `tokio::main` macro. This will cause threads to be executed concurrently using time
slicing instead threads.

## Wrapping up

In this tutorial we learned how to use the `async` and `await` keywords to write asynchronous code. We also learned how
to use the `tokio` as a runtime to execute our asynchronous code concurrently.

When you are dealing with asynchronous code you need to be aware that we are telling the runtime when a
block of asynchronous code is ready to yield so that other tasks can be executed. This gives us ore control however it
also puts more responsibility on us. We need to make sure that we are writing efficient code. For example, we don't want
to put CPU intensive operations inside an asynchronous block because this will block the thread and prevent other tasks
from being executed.

## Further reading

- https://kube.rs/
- https://tokio.rs/