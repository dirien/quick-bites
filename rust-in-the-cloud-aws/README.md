# Rust in the Cloud: Running Rust based functions in AWS

## Introduction

This blog post is all about running Rust-based functions on AWS Lambda. Will see, how much effort is required to get started and what the pros and cons are. If you are already familiar with AWS Lambdas, you may know that AWS Lambdas offers a variety of runtimes. A runtime provides a language-specific environment to run your function in. You can use this AWS provided runtimes, or you can build your own.

Now comes the interesting part for us: Because Rust is compiled to native code, we do not need a dedicated runtime to run our Rust code. We will use instead the [Rust runtime client](https://github.com/awslabs/aws-lambda-rust-runtime) to build locally (or in your CI system) and then upload the binary using the AWS CLI by defining the AWS Lambda function. As runtime, we will use `provided.al2` which is a runtime that does not include any language-specific dependencies.

The demo code? A simple function that returns as AI-generated <mark>Dad Joke</mark>. The function will give us some, hopefully, funny dad jokes.

Additionally, we cross-compile our lambda function to `arm64` architecture to profit from the [AWS Graviton](https://docs.aws.amazon.com/whitepapers/latest/aws-graviton-performance-testing/what-is-aws-graviton.html) processor. Graviton-based instances offer the best bang for your buck and are up to 40% cheaper than Intel-based solutions.

You are new to Rust and stumbled over this blog article? I have a whole series of blog posts about rust, if you are interested in more, check out my Rust 🦀 series:

%[https://blog.ediri.io/series/learning-rust] 

### Why AWS CLI?

You may ask yourself: *Why we are not using an infrastructure as code tool like to define and deploy our AWS Lambda function?*

The reason is simple: I want to keep the blog post as simple as possible. We will use [Pulumi](https://www.pulumi.com/) later in some more sophisticated blog posts. For now, we will use the AWS CLI to define our AWS Lambda function. Stay tuned for more to come.

## Prerequisites

To follow this blog post, you should have a basic understanding of Rust and the Cargo build tool. If you are new to Rust check out my blog post:

%[https://blog.ediri.io/learn-rust-in-under-10-mins] 

Before we start, we need to make sure we have the following tools installed:

* [Rust](https://www.rust-lang.org)

* An IDE or text editor of your choice

* AWS account (free tier is enough)

* AWS CLI

* ChatGPT API key (No need for ChatGPT Plus)


## Create a new Rust project

Before we can start with the rust code, we need to install the `cargo-lambda` Cargo plugin. As I am using a Mac, I will use `brew` to install the plugin.

If you are using a different OS, check out the [Getting started section](https://github.com/awslabs/aws-lambda-rust-runtime#getting-started) of the official documentation.

```bash
brew tap cargo-lambda/cargo-lambda
brew install cargo-lambda
```

Now we can create our function by running the `cargo-lambda` with the subcommand `new`.

```bash
cargo lambda new dadjoke
```

You will be asked a few questions. I have answered them as follows:

```bash
cargo lambda new dadjoke   
> Is this function an HTTP function? Yes
```

This will generate a new Rust project inside the folder named `dadjoke`.

Change into the newly created folder and add our `chatgpt_rs` crate using the `cargo add` command.

```bash
cd dadjoke
cargo add chatgpt_rs 
```

Open the `src/main.rs` file and replace the existing content with the following code, as we want to access the `ChatGPT` API to generate an AI-generated Dad Jokes for us.

```rust
use chatgpt::prelude::ChatGPT;
use chatgpt::types::CompletionResponse;
use lambda_http::{run, service_fn, Error, Request, Response};


async fn function_handler(_: Request) -> Result<Response<String>, Error> {
    let chatgpt_api_key = std::env::var("CHATGPT_API_KEY").expect("env variable `CHATGPT_API_KEY` should be set");

    let client = ChatGPT::new(chatgpt_api_key)?;

    let response: CompletionResponse = client
        .send_message("Imagine your a dad and tell now a good dad joke, tell only the joke without any other text, all in one line without line breaks:")
        .await.unwrap();

    let mut reponse_string = response.message().content.to_string();
    reponse_string = reponse_string.replace("\n", "");

    println!("{}", reponse_string);
    
    let resp = Response::builder()
        .status(200)
        .header("content-type", "text/html")
        .body(reponse_string)
        .map_err(Box::new)?;
    Ok(resp)
}

#[tokio::main]
async fn main() -> Result<(), Error> {
    tracing_subscriber::fmt()
        .with_max_level(tracing::Level::INFO)
        .with_target(false)
        .without_time()
        .init();

    run(service_fn(function_handler)).await
}
```

The main action happens in the `function_handler` function. The API key is provided via the environment variable `CHATGPT_API_KEY`. We will set this variable later when we deploy our function to AWS Lambda. We then make a non-streaming call to ChatGPT to give us a dad joke back. I use the following phrase to give the AI the right context from the start:

> Imagine your a dad and tell now a good dad joke, tell only the joke without any other text, all in one line without line breaks:

## Build and Test Our Function Locally

Before we deploy our function to AWS, we want to test it locally to be sure it works as expected. To do so, the `cargo lambda` subcommand `watch` offers us a very easy way to build our function and start a local HTTP server on the port `9000`.

```bash
export CHATGPT_API_KEY=<YOUR_CHATGPT_API_KEY>
cargo lambda watch -a 0.0.0.0 -p 9000
```

Now we can invoke our function using `cargo lambda invoke`. This will send an HTTP request to our local server and print the response with handling all the event parameters for us.

```bash
cargo lambda invoke dadjoke --data-ascii '{}' -a 0.0.0.0
```

You should see something like this:

```bash
{"statusCode":200,"headers":{"content-type":"text/html"},"multiValueHeaders":{"content-type":["text/html"]},"body":"Response: Why don't skeletons fight each other? They don't have the guts!","isBase64Encoded":false}
```

## Deploy our function to AWS Lambda

We are ready to deploy our function to AWS!

First, we need to build and bundle our function. We can do this, again, by using the `cargo lambda` with the subcommand `build`.

<div data-node-type="callout">
<div data-node-type="callout-emoji">🤓</div>
<div data-node-type="callout-text">As we want to use Graviton-based instances, we need to cross-compile our function to <code>arm64</code>. We can do this by adding <code>--arm64</code> to the <code>cargo lambda build</code> command.</div>
</div>

```bash
cargo lambda build --release --arm64 --output-format zip
```

This will create a zip file inside the `target/lambda/release` folder.

Now we can use the following AWS CLI command to create the execution role for our function and then create the function itself.

```bash
aws iam create-role --role-name rust-in-the-cloud-role --assume-role-policy-document '{"Version": "2012-10-17","Statement": [{ "Effect": "Allow", "Principal": {"Service": "lambda.amazonaws.com"}, "Action": "sts:AssumeRole"}]}'
```

After the role is created, we need to attach the `AWSLambdaBasicExecutionRole` policy to the role.

```bash
aws iam attach-role-policy --role-name rust-in-the-cloud-role --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
```

With the role in place, we can now create our function. We will use the `aws lambda create-function` command to do so.

```bash
aws lambda create-function --function-name rust-in-the-cloud \
--handler bootstrap \
--zip-file fileb://./target/lambda/dadjoke/bootstrap.zip \
--runtime provided.al2 \
--role arn:aws:iam::<YOUR_AWS_ACCOUNT_ID>:role/rust-in-the-cloud-role \
--environment Variables={CHATGPT_API_KEY=<YOUR_CHATGPT_API_KEY>} \
--tracing-config Mode=Active
```

To get the role arn, you can use the `aws iam list-roles` command.

```bash
aws iam list-roles | grep rust-in-the-cloud-role
```

Now we can use the `aws lambda invoke` command to synchronously invoke our function.

```bash
aws lambda invoke --cli-binary-format raw-in-base64-out  \
--function-name rust-in-the-cloud \
--payload '{}' response.json
```

Now we can use the `jq` command to extract the response from the `response.json` file.

```bash
cat response.json | jq '.body'
```

You should see something like this:

```bash
"Why don't eggs tell jokes? Because they might crack up!"
```

## Conclusion

We have seen how easy it is to run Rust based functions in AWS Lambda. With the help of the `cargo-lambda` plugin, we can build and bundle our function and then use for example the AWS CLI to deploy it to AWS Lambda.

Let's have a look at the pros and cons of this approach:

### Pros

* Easy to get started

* Lightning fast cold start times and low memory footprint

* No need to build a custom runtime


But there are also some cons:

### Cons

* I have the feeling that AWS is not really supporting Rust as a first class citizen.

* The documentation is not that great, and you need to figure some stuff out by yourself.


Overall, I am still very happy using Rust in AWS Lambda and I could see adding Rust based functions to a polyglot microservice architecture in the future.

If you want to learn more, check out the official documentation [here](https://docs.aws.amazon.com/lambda/latest/dg/lambda-rust.html).

## Housekeeping

To clean up, we can delete the function and the role.

```bash
aws lambda delete-function --function-name rust-in-the-cloud
aws iam detach-role-policy --role-name rust-in-the-cloud-role --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
aws iam delete-role --role-name rust-in-the-cloud-role
```
