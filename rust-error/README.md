# panic! with sense: Error handling in Rust 🦀

## Introduction

In this article, I will discuss error handling in `Rust` 🦀. I try to explain the differences between recoverable and unrecoverable errors, and how to handle them properly in your code.

At the end I will also talk about two popular crates for error handling in `Rust` 🦀: `anyhow` and `thiserror`.

## The Panic Macro and Unrecoverable Errors

A `Panic` is an exception that a `Rust` 🦀 program can throw. It stops all execution in the current thread. Panic, will return a short description of the error and the location of the panic in the source code.

Let's look at an example:

```rust
fn main() {
    println!("Hello, world!");
    panic!("oh no!");
}
```

This will print `Hello, world!` and then panic with the message `oh no!` and the location of the panic in the source code.

If your running this code in a terminal, you will see the following output:

```bash
cargo run                                                
   Compiling rust-error v0.1.0 (/Users/dirien/Tools/repos/quick-bites/rust-error)
    Finished dev [unoptimized + debuginfo] target(s) in 0.61s
     Running `target/debug/rust-error`
Hello, world!
thread 'main' panicked at 'oh no!', src/main.rs:3:5
note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace
```

The message gives us also a hint on how to display a backtrace. If you run the code with the environment variable `RUST_BACKTRACE=1` you will get a list of all the functions leading up to the panic.

```bash
RUST_BACKTRACE=1 cargo run
    Finished dev [unoptimized + debuginfo] target(s) in 0.01s
     Running `target/debug/rust-error`
Hello, world!
thread 'main' panicked at 'oh no!', src/main.rs:3:5
stack backtrace:
   0: rust_begin_unwind
             at /rustc/897e37553bba8b42751c67658967889d11ecd120/library/std/src/panicking.rs:584:5
   1: core::panicking::panic_fmt
             at /rustc/897e37553bba8b42751c67658967889d11ecd120/library/core/src/panicking.rs:142:14
   2: rust_error::main
             at ./src/main.rs:3:5
   3: core::ops::function::FnOnce::call_once
             at /rustc/897e37553bba8b42751c67658967889d11ecd120/library/core/src/ops/function.rs:248:5
note: Some details are omitted, run with `RUST_BACKTRACE=full` for a verbose backtrace.
```

In this case the backtrace is not very useful, because the panic is in the `main` function.

Let's look at a different example, which is extremely contrived, but for demonstration purposes it will do.

```rust
fn a() {
    b();
}

fn b() {
    c("engin");
}

fn c(name: &str) {
    if name == "engin" {
        panic!("Dont pass engin");
    }
}

fn main() {
    a();
}
```

We have three functions `a`, `b` and `c`. The main function calls `a`. `a` calls `b` and `b` calls `c`. `c` takes a string as an argument and panics if the string is `engin`.

```bash
cargo run
   Compiling rust-error v0.1.0 (/Users/dirien/Tools/repos/quick-bites/rust-error)
    Finished dev [unoptimized + debuginfo] target(s) in 0.14s
     Running `target/debug/rust-error`
thread 'main' panicked at 'Dont pass engin', src/main.rs:11:9
note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace
```

This error is not very useful. We can see that the panic happened in `c`, but we don't know which function called `c`.

If we run the code with the environment variable `RUST_BACKTRACE=1` we get the following output:

```bash
RUST_BACKTRACE=1 cargo run
    Finished dev [unoptimized + debuginfo] target(s) in 0.01s
     Running `target/debug/rust-error`
thread 'main' panicked at 'Dont pass engin', src/main.rs:11:9
stack backtrace:
   0: rust_begin_unwind
             at /rustc/897e37553bba8b42751c67658967889d11ecd120/library/std/src/panicking.rs:584:5
   1: core::panicking::panic_fmt
             at /rustc/897e37553bba8b42751c67658967889d11ecd120/library/core/src/panicking.rs:142:14
   2: rust_error::c
             at ./src/main.rs:11:9
   3: rust_error::b
             at ./src/main.rs:6:5
   4: rust_error::a
             at ./src/main.rs:2:5
   5: rust_error::main
             at ./src/main.rs:16:5
   6: core::ops::function::FnOnce::call_once
             at /rustc/897e37553bba8b42751c67658967889d11ecd120/library/core/src/ops/function.rs:248:5
note: Some details are omitted, run with `RUST_BACKTRACE=full` for a verbose backtrace.
```

This is much better. We can see that the panic happened in `c`, and we can see the call stack leading up to the panic. We see that `c` was called by `b`, which was called by `a`, which was called by `main`. So let's change the code in `b` to call `c` with a different name.

```rust
fn b() {
    c("dirien");
}
```

Now the code compiles and runs without any problems.

## Recoverable Errors

A recoverable error is an error that can be handled by the code. For example, if we try to open a file that does not exist, we can handle the error and print a message to the user or create the file instead crashing the program.

For this case we can use the `Result` type. The `Result` type is an enum with two variants: `Ok` and `Err`. The `Ok` variant indicates that the operation was successful and stores a generic value. The `Err` variant indicates that the operation failed and stores an error value.

Like the `Option` type, the `Result` type is defined in the standard library, and we need to bring it into scope.

Let's look at an example. We will try to open a file and read the contents of the file.

```rust
fn main() {
    let f = File::open("hello.txt");
}
```

Here we need to check the result of the `open` function. If the file is opened successfully, we can read the contents of the file. If the file is not opened successfully, we can print an error message to the user.

To check the result of the `open` function, we can use the `match` expression. The `match` expression is similar to the `if` expression, but it can handle more than two cases. We're also shadowing the `f` variable and setting it to the `match` expression.

```rust
fn main() {
    let f = File::open("hello.txt");

    let f = match f {
        Ok(file) => file,
        Err(error) => panic!("There was a problem opening the file: {:?}", error),
    };
}
```

If the `open` function returns `Ok`, we store the file handle in the `f` variable. If the `open` function returns `Err`, we panic and print the error message.

Let us run the code and see what happens.

```bash
cargo run
warning: `rust-error` (bin "rust-error") generated 1 warning
    Finished dev [unoptimized + debuginfo] target(s) in 0.00s
     Running `target/debug/rust-error`
thread 'main' panicked at 'There was a problem opening the file: Os { code: 2, kind: NotFound, message: "No such file or directory" }', src/main.rs:7:23
note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace
```

We get a panic, but the error message is much more useful. We can see that the error is `Os { code: 2, kind: NotFound, message: "No such file or directory" }`. This error makes sense, because we are trying to open a file that does not exist.

Now let's enhance the code to instead of panicking, we will create the file if it does not exist. First we will bring the `ErrorKind` enum into scope.

```rust
use std::fs::File;
use std::io::ErrorKind;
...
```

Then we will use the `match` expression to check the error kind. If the error kind is `NotFound`, we will create the file. But the creation of the file can also fail, so we will use the `match` expression again to check the result of the `create` function. If the `create` function returns `Ok`, we will return the file handle. If the `create` function returns `Err`, we will panic.

The last part is to use `other_error` to handle all other errors that are not `ErrorKind::NotFound`.

```rust
fn main() {
    let f = File::open("hello.txt");

    let f = match f {
        Ok(file) => file,
        Err(error) => match error.kind() {
            ErrorKind::NotFound => match File::create("hello.txt") {
                Ok(fc) => fc,
                Err(e) => panic!("Problem creating the file: {:?}", e),
            },
            other_error => panic!("There was a problem opening the file: {:?}", other_error),
        },
    };
}
```

Now when we run the code, we can see that no panic happens. And if we check the directory, we can see that the file was created.

```bash
cargo run
   Compiling rust-error v0.1.0 (/Users/dirien/Tools/repos/quick-bites/rust-error)
    Finished dev [unoptimized + debuginfo] target(s) in 0.62s
     Running `target/debug/rust-error`
```

But this code is not very readable. We have a lot of `match` expressions. A better way to handle this is to use closures. We will use closures to handle the `Ok` and `Err` variants of the `Result` type.

When we attempt to open a file, we will use the `unwrap_or_else` method which gives us back the file or calls the anonymous function or closure that we pass the error to. Inside the closure, we will check the error kind. If the error is `NotFound` then we attempt to create the file call the `unwrap_or_else` method again. This gives us back the file if the calls succeeds. Note that we don't have a semicolon at the end which means this is an expression and not a statement. In the failure case we have another closure that will just panic.

```rust
fn main() {
    let f = File::open("hello.txt").unwrap_or_else(|error| {
        if error.kind() == ErrorKind::NotFound {
            File::create("hello.txt").unwrap_or_else(|error| {
                panic!("Problem creating the file: {:?}", error);
            })
        } else {
            panic!("There was a problem opening the file: {:?}", error);
        }
    });
}
```

Now we going to rewrite the code again to use the `unwrap` and `expect` methods. The unwrap method is a shortcut method that is implemented on `Result` types. If the `Result` is `Ok`, the `unwrap` method will return the value inside the `Ok`. If the `Result` is `Err`, the `unwrap` method will call the `panic!` macro for us.

```rust
fn main() {
    let f = File::open("hello.txt").unwrap();
}
```

When we run the code, we get the same error as before.

```bash
cargo run
    Finished dev [unoptimized + debuginfo] target(s) in 0.10s
     Running `target/debug/rust-error`
thread 'main' panicked at 'called `Result::unwrap()` on an `Err` value: Os { code: 2, kind: NotFound, message: "No such file or directory" }', src/main.rs:4:37
note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace
```

The `expect` method is similar to the `unwrap` method, but we can pass a custom error message to the `expect` method. This error message will be printed when the `Result` is `Err`.

```rust
fn main() {
    let f = File::open("hello.txt").expect("OMG! I cant open the file!");
}
```

When we run the code, we can see our custom error message.

```bash
cargo run
   Compiling rust-error v0.1.0 (/Users/dirien/Tools/repos/quick-bites/rust-error)
    Finished dev [unoptimized + debuginfo] target(s) in 0.10s
     Running `target/debug/rust-error`
thread 'main' panicked at 'OMG! I cant open the file!: Os { code: 2, kind: NotFound, message: "No such file or directory" }', src/main.rs:4:37
note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace
```

## How to propagate errors

In the previous section we saw how to handle errors. But what if we want to propagate the error to the caller of our function? This gives the caller the ability to handle the error.

Let's say we want to read the contents of a file. We will create a function that reads username from a file. The function will return a `Result` type. The `Result` type will contain a `String` on success and the `io::Error` on error.

If the file does not exist, we will return the error. If the file exists, we will try to read the contents of the file. If this not successful, we will return the error. If the read is successful, we will return the username.

```rust
use std::fs::File;
use std::io::{self, Read};

fn read_username_from_file() -> Result<String, io::Error> {
    let username_file_result = File::open("hello.txt");

    let mut username_file = match username_file_result {
        Ok(file) => file,
        Err(e) => return Err(e),
    };

    let mut username = String::new();

    match username_file.read_to_string(&mut username) {
        Ok(_) => Ok(username),
        Err(e) => Err(e),
    }
}
```

We can shorten the code by using the `?` operator. The `?` operator can only be used in functions that return a `Result` type. The `?` operator is similar to our `unwrap` and `expect` methods. If the `Result` is `Ok`, the `?` operator will return the value inside the `Ok`. If the `Result` is `Err`, instead of calling the `panic!` macro, the `?` operator will return the error and early exit the function.

If everything is successful, the `?` operator will return the safely return the value inside the `Ok`.

```rust
use std::fs::File;
use std::io::{self, Read};

fn read_username_from_file() -> Result<String, io::Error> {
    let mut username_file = File::open("hello.txt")?;

    let mut username = String::new();

    username_file.read_to_string(&mut username)?;

    Ok(username)
}
```

We can shorten the code even more by chaining method calls. The `?` operator can be used with method calls that return a `Result` type.

```rust
use std::fs::File;
use std::io::{self, Read};

fn read_username_from_file() -> Result<String, io::Error> {
    let mut username = String::new();

    File::open("hello.txt")?.read_to_string(&mut username)?;

    Ok(username)
}
```

But we can make the code even shorter by using the system module function `fs::read_to_string`. The `fs::read_to_string` function will open the file, create a new `String`, read the contents of the file into the `String`, and return it. If any of these steps fail, the `fs::read_to_string` function will return the error.

```rust
use std::fs;
use std::io;

fn read_username_from_file() -> Result<String, io::Error> {
    fs::read_to_string("hello.txt")
}
```

As mentioned before, the `?` operator can only be used in functions that return a `Result` type. If we want to use the `?` operator in the `main` function, we have to change the return type of the `main` function to `Result`. The `main` function can also return a `Result` type.

```rust
use std::error::Error;
use std::fs::File;

fn main() -> Result<(), Box<dyn Error>> {
    let greeting_file = File::open("hello.txt")?;
    Ok(())
}
```

The `main` function returns a `Result` type. The `Result` type contains a `()` on success and a `Box<dyn Error>` on error.

## Error helper crates

There are a lot of crates that can help you with error handling. In this section we will look at the `anyhow` crate and the `thiserror` crate. This is not an exhaustive list of error handling crates, but it will give you an idea of what is out there.

Of course, we can not go to deep into these crates. If you want to learn more about these crates, you can check out the links at the end of this section.

### The `thiserror` crate

`thiserror` provides a derived implementation which adds the `error` trait for us. This makes it easier to implement the `error` trait for our custom error types.

To use the `thiserror` crate, we have to add the crate to our `Cargo.toml` file. This `cargo add` command will add the `thiserror` crate to our `Cargo.toml` file.

```bash
cargo add thiserror
```

We can now use the `thiserror` crate in our code. We will create a custom error type for our `read_username_from_file` function called `CustomError`.

```rust
use std::error::Error;
use std::fs::File;
use std::io::Read;

#[derive(Debug, thiserror::Error)]
enum CustomError {
    #[error("OMG! There is an error {0}")]
    BadError(#[from] std::io::Error),

}

fn read_username_from_file() -> Result<String, CustomError> {
    let mut username = String::new();
    File::open("hello.txt")?.read_to_string(&mut username)?;
    Ok(username)
}
```

### The `anyhow` crate

`anyhow` provides an idiomatic alternative to explicitly handling errors. It is similar to the previously mentioned `error` trait but has additional features such as adding context to thrown errors.

To add the `anyhow` crate to our project, we can use the `cargo add` command.

```bash
cargo add anyhow
```

We can now use the `anyhow` crate in our code. We will create a custom error type for our `read_username_from_file` function called `CustomError`.

```rust
use std::fs::File;
use std::io::Read;
use anyhow::Context;


fn read_username_from_file() -> Result<String, anyhow::Error> {
    let mut username = String::new();

    File::open("hello.txt").context("Failed to open file")?.read_to_string(&mut username).context("Failed to read file")?;

    Ok(username)
}
```

### When to use `thiserror` and `anyhow`

The `thiserror` crate is useful when you want to implement the `Error` trait for your custom error types. The `anyhow` crate is useful when you don't care about the error type and just want to add context to the error.

## Summary

In this article we looked at error handling in `Rust` 🦀. We talked about non-recoverable errors and recoverable errors. The error handling in `Rust` 🦀 is designed to help you in writing code that is more robust and less error-prone. The `panic!` macro is used for non-recoverable errors when your program is in a state where it can not continue and should stop instead of trying to proceed with invalid or incorrect data. The `Result` type is used for recoverable errors. The `Result` enums indicates that the operation can fail and that our code can recover from the error and the caller of the piece of code has to handle the success or failure of the operation.

## Resources

- [Error Handling](https://doc.rust-lang.org/book/ch09-00-error-handling.html)
- [The `anyhow` crate](https://docs.rs/anyhow/latest/anyhow/)
- [The `thiserror` crate](https://docs.rs/thiserror/latest/thiserror/)
