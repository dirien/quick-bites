# 🥊 Fiber (Go) vs. Nickel.rs (Rust) 🥊: A Framework Showdown in 'Hello World'

## Introduction

In this article, I want to compare the performance of two different web frameworks for Rust and Go. Both frameworks are
very similar in their design (all are inspired by Express.js) and both claim to be the fastest web framework (blazing
fast). On top of that both frameworks are easy to use, which is a big plus for me (I am not the smartest guy in the
world, so easy is good).

## The Competitors in detail

### Nickel.rs (Rust)

For Rust, I have chosen the [Nickel.rs](https://nickel-org.github.io/) framework. It is a minimal and lightweight
framework for web apps in Rust. It is inspired by Express.js and provides a lot of features like flexible routing,
middleware, JSON handling, and more.

### Fiber (Go)

As contender for Go, I have chosen the [Fiber](https://gofiber.io/) framework. This framework is also inspired by
Express.js and is build on top of [Fasthttp](https://github.com/valyala/fasthttp). It has a lot of features like
middleware, routing, websockets, and more. On top it claims to have extreme performance and a small memory footprint.

## The test

The specs of my machine are: Apple M1 Max (10 Core CPU) with 32GB of RAM.

The Tests will be written in [bombardier](https://github.com/codesenberg/bombardier) and will be executed for 50, 100
and 500 concurrent users with executing 5M requests.

I use following versions:

- Go: go1.20.3 darwin/arm64
- Rust: rustc 1.65.0 (897e37553 2022-11-02)

### The test code

#### `Nickel.rs (Rust)`

```rust 
#[macro_use]
extern crate nickel;

use nickel::Nickel;

fn main() {
    let mut server = Nickel::new();

    server.utilize(router! {
        get "**" => |_req, _res| {
            "Hello world!"
        }
    });

    server.listen("127.0.0.1:6767").unwrap();
}
```

### `Fiber (Go)`

```go
package main

import "github.com/gofiber/fiber/v2"

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World 🐹!")
	})

	app.Listen(":3000")
}
```

### The results

#### 50 concurrent users

|                      | `Fiber (Go)` | `Nickel.rs (Rust)` |
|----------------------|--------------|--------------------|
| Time taken           | 140s         | 47s                |
| Request per second   | 35378.09     | 106293.29          |
| Mean response time   | 1.41 ms      | 0.39396 ms         |
| Median response time | 0.845 ms     | 0.049 ms           |
| 90th percentile      | 3.44 ms      | 0.110 ms           |
| Max response time    | 107.07 ms    | 33.91 s            |
| CPU                  | 35%          | 18%                |
| Memory               | 11.102 MB    | 4 MB               |

#### 100 concurrent users

|                      | `Fiber (Go)` | `Nickel.rs (Rust)` |
|----------------------|--------------|--------------------|
| Time taken           | 184s         | 51s                |
| Request per second   | 27073.55     | 97200.03           |
| Mean response time   | 3.69 ms      | 0.92 ms            |
| Median response time | 2.90 ms      | 0.04 ms            |
| 90th percentile      | 7.94 ms      | 0.09400 ms         |
| Max response time    | 136.02 ms    | 29.91 s            |
| CPU                  | 35%          | 18%                |
| Memory               | 13 MB        | 4 MB               |

#### 500 concurrent users

|                      | `Fiber (Go)` | `Nickel.rs (Rust)` |
|----------------------|--------------|--------------------|
| Time taken           | 189s         | 1m                 |
| Request per second   | 26359.05     | 83084.80           |
| Mean response time   | 18.97 ms     | 5.00 ms            |
| Median response time | 18.27 ms     | 0.04 ms            |
| 90th percentile      | 31.78 ms     | 0.085 ms           |
| Max response time    | 185.04 ms    | 32.25 s            |
| CPU                  | 35%          | 17%                |
| Memory               | 29 MB        | 4 MB               |

## Conclusion 🎉

Based on the data provided, `Nickel.rs (Rust)` is the winner. There are several reasons for this:

1. 🚀 Faster response times: Nickel.rs (Rust) has lower mean, median, and 90th percentile response times across all
   levels of concurrent users compared to Fiber (Go).
1. ⚡ Higher request per second: Nickel.rs (Rust) can handle more requests per second than Fiber (Go) in each test.
1. 🌡️ Lower CPU usage: Nickel.rs (Rust) uses less CPU (about half) compared to Fiber (Go) in all tests.
1. 🧠 Lower memory usage: Nickel.rs (Rust) uses significantly less memory (about 3 to 7 times less) compared to Fiber (
   Go) in all tests.

In conclusion, 🏆 Rust (specifically `Nickel.rs`) outperforms Go (`Fiber`) in terms of response times, request handling,
CPU usage, and memory consumption, making it the winner in this comparison.
