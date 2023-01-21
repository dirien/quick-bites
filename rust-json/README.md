# Serialize and Deserialize Data in Rust using serde and serde_json

## Introduction

In this blog post, I will show you how to serialize and deserialize data in `Rust 🦀` using the `serde` library. We will also take a look into the `serde_json` library to serialize and deserialize JSON data in `Rust 🦀`. To give our demo application more "value", we will talk to a REST API and serialize and deserialize the data we get from the API. For this, we will add the `reqwest` library to our project. More about this in the next section.

## Prerequisites

To follow this blog post, you should have a basic understanding of `Rust 🦀` and the Cargo build tool. If you are new to `Rust 🦀` check out my blog post "Learn Rust in under 10 mins":

%[https://blog.ediri.io/learn-rust-in-under-10-mins] 

Before we start, we need to make sure we have the following tools installed:

* [Rust](https://www.rust-lang.org)

* An IDE or text editor of your choice


## Initialize the demo project

```bash
cargo init
```

This will create a new `Rust 🦀` project in the current directory. Now, we need to add the dependencies we need for our demo application.

As I mentioned before, we will use the `reqwest` library to talk to a REST API. The `reqwest` library provides a convenient, high-level client build on the `hyper` library. By default `reqwest` includes a client that is able of making asynchronous requests. But you can also use the `reqwest` library in a synchronous way if you don't want the added complexity of asynchronous code. For this demo application, we will use the asynchronous client and will need to add `tokio` to our project. `tokio` is a runtime for asynchronous `Rust 🦀` applications. I wrote a blog post about `async/await` in `Rust 🦀`, feel free to check it out too if you are new to `async/await`:

%[https://blog.ediri.io/how-to-asyncawait-in-rust-an-introduction] 

Now add the dependencies to our `Cargo.toml` file using the `cargo add` command:

```bash
cargo add reqwest --features="json"
cargo add serde --features="derive"
cargo add tokio --features="full"
```

## Talk to a REST API

Now that we have all the dependencies we need, we can start writing some code. First, we set up the code to talk to a REST API. For this demo application, we will use the [DummyJSON](https://dummyjson.com/) REST API. The `DummyJSON` API gives us great dummy JSON data to use as a placeholder in development.

In the `src/main.rs` we first set up the `main` function and wiring `tokio` to our application to handle asynchronous code:

```rust
#[tokio::main]
async fn main() {
    println!("Hello, world!");
}
```

We test our application by running `cargo run` in the terminal. This should print `Hello, world!` to the terminal.

```bash
➜ cargo run 
    Finished dev [unoptimized + debuginfo] target(s) in 0.05s
     Running `target/debug/rust-json`
Hello, world!
```

Now we can add the code to talk to the `DummyJSON` API. First, we create a new `reqwest` client and then we make a GET request to the `DummyJSON` product API. We use `await` to wait for the response and then turn the response into a string also `await`ing this operation. After all this, we save the string into a variable called `product` and finally, print it to the terminal using the `println!` macro

> As we used the `await` method which will return a future, we need to handle possible errors in the `main` function too. The `main` function will return a `Result` type and a `reqwest::Error` if something goes wrong.

Check this my blog about error handling in `Rust 🦀`, for more information about this topic.

%[https://blog.ediri.io/panic-with-sense-error-handling-in-rust] 

The updated `main` function looks like this:

```rust
use reqwest::{Client, Error};

#[tokio::main]
async fn main() -> Result<(), Error> {
    let product = Client::new()
        .get("https://dummyjson.com/products/1")
        .send()
        .await?
        .text()
        .await?;
    println!("{:#?}", product);
    Ok(())
}
```

If we run the application now, we should see the JSON data we get from the DummyJSON API in the terminal:

```bash
➜ cargo run 
   Compiling rust-json v0.1.0 (/Users/dirien/Tools/repos/quick-bites/rust-json)
    Finished dev [unoptimized + debuginfo] target(s) in 0.47s
     Running `target/debug/rust-json`
"{\"id\":1,\"title\":\"iPhone 9\",\"description\":\"An apple mobile which is nothing like apple\",\"price\":549,\"discountPercentage\":12.96,\"rating\":4.69,\"stock\":94,\"brand\":\"Apple\",\"category\":\"smartphones\",\"thumbnail\":\"https://i.dummyjson.com/data/products/1/thumbnail.jpg\",\"images\":[\"https://i.dummyjson.com/data/products/1/1.jpg\",\"https://i.dummyjson.com/data/products/1/2.jpg\",\"https://i.dummyjson.com/data/products/1/3.jpg\",\"https://i.dummyjson.com/data/products/1/4.jpg\",\"https://i.dummyjson.com/data/products/1/thumbnail.jpg\"]}"
```

Of course, we don't want to print the JSON data just to the terminal. We want to deserialize the JSON data into a `Rust 🦀` struct.

## Serialize and Deserialize JSON data

To deserialize the response from the DummyJSON API into a `Rust 🦀` struct, we need to create a struct called `Product` that matches the JSON data we get from the API.

The first thing to do is to import the `Serialize` and `Deserialize` traits from the `serde` library:

```rust
use serde::{Deserialize, Serialize};
```

Now we can create the `Product` struct and add the `Serialize` and `Deserialize` traits to it:

```rust
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Product {
    pub id: i64,
    pub title: String,
    pub description: String,
    pub price: i64,
    pub rating: f64,
    pub stock: i64,
    pub brand: String,
    pub category: String,
    pub thumbnail: String,
    pub images: Vec<String>,
}
```

As the API returns the JSON data in camelCase, we need to add the `#[serde(rename_all = "camelCase")]` attribute to the struct as in `Rust 🦀` by convention, the struct fields are in snake case.

Now that we have the `Product` struct, we can deserialize the JSON data into a `Product` struct. Instead of converting the response to a string, we can convert it directly to a `Product` struct. We do this by using the `json` method on the response and then `await`ing the result. The `json` method will deserialize the JSON data into a `Product` struct. We explicitly annotate the type of the variable `product` to be a `Product` struct.

```rust
#[tokio::main]
async fn main() -> Result<(), Error> {
    let product: Product = Client::new()
        .get("https://dummyjson.com/products/1")
        .send()
        .await?
        .json()
        .await?;
    println!("{:#?}", product);
    Ok(())
}
```

If we run the application now, we should see the `Product` struct in the terminal:

```bash
➜ cargo run 
   Compiling rust-json v0.1.0 (/Users/dirien/Tools/repos/quick-bites/rust-json)
    Finished dev [unoptimized + debuginfo] target(s) in 1.14s
     Running `target/debug/rust-json`
Product {
    id: 1,
    title: "iPhone 9",
    description: "An apple mobile which is nothing like apple",
    price: 549,
    rating: 4.69,
    stock: 94,
    brand: "Apple",
    category: "smartphones",
    thumbnail: "https://i.dummyjson.com/data/products/1/thumbnail.jpg",
    images: [
        "https://i.dummyjson.com/data/products/1/1.jpg",
        "https://i.dummyjson.com/data/products/1/2.jpg",
        "https://i.dummyjson.com/data/products/1/3.jpg",
        "https://i.dummyjson.com/data/products/1/4.jpg",
        "https://i.dummyjson.com/data/products/1/thumbnail.jpg",
    ],
}
```

Now we can change our code to serialize a `Product` struct to JSON data and post it to the DummyJSON API `create product` endpoint.

We need to create a `Product` struct and fill it with some data. We then need to create a new `reqwest::Client` but this time we use `post` instead of `get` method. We then set the body of the request to JSON and pass a reference to our newly created `Product` struct. We then `await` the response and turn the response into `json` and `await` the result. The result will then be saved in the `new_product` variable to finally print it to the terminal.

```rust
#[tokio::main]
async fn main() -> Result<(), Error> {
    let product: Product = Client::new()
        .get("https://dummyjson.com/products/1")
        .send()
        .await?
        .json()
        .await?;
    println!("{:#?}", product);


    let new_product = Product {
        id: 1,
        title: "Macbook Pro".to_owned(),
        description: "Best laptop ever".to_owned(),
        price: 100,
        rating: 0.0,
        stock: 100,
        brand: "Apple".to_owned(),
        category: "laptops".to_owned(),
        thumbnail: "https://dummyimage.com/300x300/000/fff".to_owned(),
        images: vec![
            "https://dummyimage.com/300x300/000/fff".to_owned(),
            "https://dummyimage.com/600x600/000/fff".to_owned(),
        ],
    };

    let new_product: Product = Client::new()
        .post("https://dummyjson.com/products/add")
        .header("Content-Type", "application/json")
        .json(&new_product)
        .send()
        .await?
        .json()
        .await?;
    println!("{:#?}", new_product);
    Ok(())
}
```

If we run the application now, we should see our new Product successfully created with a new ID (`101`)

```bash
Product {
    id: 101,
    title: "Macbook Pro",
    description: "Best laptop ever",
    price: 100,
    rating: 0.0,
    stock: 100,
    brand: "Apple",
    category: "laptops",
    thumbnail: "https://dummyimage.com/300x300/000/fff",
    images: [
        "https://dummyimage.com/300x300/000/fff",
        "https://dummyimage.com/600x600/000/fff",
    ],
}
```

That works fine, we saw how to serialize and deserialize JSON data in `Rust 🦀`.

But we are not finished as I want to show is how to handle arbitrary JSON data.

## Handling arbitrary JSON data

To start with, add the `serde_json` crate to our project:

```bash
cargo add serde_json
```

`serde_json` gives us access to a macro called `serde_json::json!` which allows using regular JSON syntax to create arbitrary JSON data.

We can add a new `reqwest::Client` to our application and instead use the `Product` struct, will switch to the `serde_json::json!` macro.

We will also change the explicit type annotation of the `product` variable to `serde_json::Value` instead of the `Product` struct. `serde_json::Value` will represent an arbitrary valid JSON value.

```rust
#[derive(Debug, Serialize, Deserialize)]
#[serde(rename_all = "camelCase")]
pub struct Product {
    pub id: i64,
    pub title: String,
    pub description: String,
    pub price: i64,
    pub rating: f64,
    pub stock: i64,
    pub brand: String,
    pub category: String,
    pub thumbnail: String,
    pub images: Vec<String>,
}


#[tokio::main]
async fn main() -> Result<(), Error> {
    // omitted previous code for brevity

    let new_product_arbitary_json: Value = Client::new()
        .post("https://dummyjson.com/products/add")
        .header("Content-Type", "application/json")
        .json(&json!({
            "id": 1,
            "title": "Macbook Pro",
            "description": "Best laptop ever",
            "price": 100,
            "rating": 0.0,
            "stock": 100,
            "brand": "Apple",
            "category": "laptops",
            "thumbnail": "https://dummyimage.com/300x300/000/fff",
            "images": [
                "https://dummyimage.com/300x300/000/fff",
                "https://dummyimage.com/600x600/000/fff"
            ]
        }))
        .send()
        .await?
        .json()
        .await?;

    println!("{:#?}", new_product_arbitary_json);
    Ok(())
}
```

With `serde_json` you will get different `serde_json:from_*` methods to parse JSON data, for example, we can use the `serde_json::from_str` function to parse a JSON string into structured data.

```rust
#[tokio::main]
async fn main() -> Result<(), Error> {
    // omitted previous code for brevity

    let test: Product = serde_json::from_value(new_product_arbitary_json).unwrap();
    println!("{:#?}", test);
    Ok(())
}
```

If you have arbitrary JSON data of type `serde_json::Value`, you can access the data using the `Value:get` method. This is very similar to `HashMap::get` method or `Vec::get` method. The `Value::get` method will return an `Option` since the key might not exist in the JSON data.

```rust

#[tokio::main]
async fn main() -> Result<(), Error> {
    // omitted previous code for brevity

    let cloned_product = new_product_arbitary_json.clone();

    let test: Product = serde_json::from_value(new_product_arbitary_json).unwrap();
    println!("{:#?}", test);

    let description = cloned_product.get("description");
    match description {
        Some(d) => println!("Description: {}", d),
        None => println!("No description"),
    }
    Ok(())
}
```

If we run the application now, we should see the following output:

```bash
# omitted previous output for brevity
Description: "Best laptop ever"
```

## Wrapping up

Hope you enjoyed this tutorial. We saw how to use `reqwest` to make HTTP requests and how to serialize and deserialize JSON data in `Rust 🦀` using `serde`. We also tried out how to handle arbitrary JSON data using `serde_json`.
