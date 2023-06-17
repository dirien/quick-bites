# How to build an SSH client using Rust

## Introduction

In this tutorial we will build a simple SSH client using Rust. Having a way to connect to a remote server is often a
requirement for DevOps tools.

If you already know Go, you might have used the [ssh](https://godoc.org/golang.org/x/crypto/ssh) package. Here is a
refresh of how to use it, with the password authentication method:

```go
package main

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

func main() {
	// SSH connection parameters
	host := "192.168.64.5"
	port := 22
	user := "steve"
	password := "megasecret"

	// Create SSH client configuration
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the remote server
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil {
		log.Fatalf("Failed to connect to SSH server: %v", err)
	}
	defer conn.Close()

	// Create a new SSH session
	session, err := conn.NewSession()
	if err != nil {
		log.Fatalf("Failed to create SSH session: %v", err)
	}
	defer session.Close()

	// Set the output writer for session's output
	session.Stdout = os.Stdout

	// Run a command on the remote server
	err = session.Run("ls -l")
	if err != nil {
		log.Fatalf("Failed to run command on remote server: %v", err)
	}
}
```

Easy, right? Now let's see how to do the same thing in Rust.

## Prerequisites

To follow this blog post, you should have a basic understanding of `Rust 🦀` and the Cargo build tool. If you are new
to `Rust 🦀` check out my blog post "Learn Rust in under 10 mins":

%[https://blog.ediri.io/learn-rust-in-under-10-mins] 

Before we start, we need to make sure we have the following tools installed:

* [Rust](https://www.rust-lang.org)

* An IDE or text editor of your choice

## Initialize the demo project

```bash
cargo init
```

This will create a new `Rust 🦀` project in the current directory. Now, we need to add the dependencies we need for our
demo application.

Now add can add the `ssh2` crate to the `Cargo.toml` file using the following command:

```bash
cargo add ssh2 --features vendored-openssl
```

## Create the SSH client

With the dependencies in place, we can now head over to the `src/main.rs` file and start writing our SSH client.

```rust
use std::io::Read;
use std::net::TcpStream;
use ssh2::Session;

fn main() {
    let stream = TcpStream::connect(format!("{}:22", "192.168.64.5"));
    match stream {
        Ok(stream) => {
            println!("Connected to the server!");
            let session = Session::new();
            match session {
                Ok(mut session) => {
                    session.set_tcp_stream(stream);
                    session.handshake().unwrap();
                    let auth = session.userauth_password("steve", "password");
                    match auth {
                        Ok(_) => {
                            println!("Authenticated!");
                            let channel = session.channel_session();
                            match channel {
                                Ok(mut channel) => {
                                    channel.exec("whoami").unwrap();
                                    let mut s = String::new();
                                    channel.read_to_string(&mut s).unwrap();
                                    println!("{}", s);
                                    channel.wait_close().unwrap();
                                    let exit_status = channel.exit_status().unwrap();
                                    if exit_status != 0 {
                                        eprint!("Exited with status {}", exit_status);
                                    }
                                }
                                Err(e) => {
                                    eprint!("Failed to create channel: {}", e);
                                }
                            }
                        }
                        Err(e) => {
                            eprint!("Failed to authenticate: {:?}", e);
                        }
                    }
                }
                Err(e) => {
                    eprint!("Failed to create session: {}", e);
                }
            }
        }
        Err(e) => {
            eprint!("Failed to connect: {}", e);
        }
    }
}
```

As you can see from the code, we try(!) to not use the `unwrap()` method as much as possible. Instead, we handle the
errors explicitly. This is a good practice to follow in your own code, to avoid `unwrap()` calls all over the place, as
it can mask errors and make your code less readable.

But let's go through the code step by step:

- Initially, a new TCP stream is established to the remote server via the `TcpStream::connect()` method. Upon successful
  connection, a new `Session` object is created.
- Subsequently, this TCP stream is set on the Session object utilizing `Session::set_tcp_stream()`.
- We then carry out the SSH handshake through `Session::handshake()`.
- After the handshake, an attempt at authentication is made using `Session::userauth_password()`. Alternatively,
  authentication could be done using a private key with `Session::userauth_pubkey_file()`.
- Finally, a new channel is formed with `Session::channel_session()`, followed by the execution of the whoami command on
  the remote server using `Channel::exec()`. The output of this command is read into a string via `Channel::
  read_to_string()` and subsequently printed to the console.
- The channel is then closed with `Channel::wait_close()` and the exit status of the command is printed to the console
  using `Channel::exit_status()`.
- If the exit status is not 0, an error message is printed to the console.

## Run the demo

Now that we have our SSH client ready, we can run it using the following command:

```bash
cargo run
```

If everything went well, you should see the following output:

```bash
Connected to the server!
Authenticated!
steve
```

## Conclusion

In this blog post, we have seen how quickly we can create an SSH client in `Rust 🦀` using the `ssh2` crate. Now we can
start building our own SSH client applications in `Rust 🦀`!




