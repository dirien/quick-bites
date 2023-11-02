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
    // Return something that implements IntoResponse.
    // It will be serialized to the right response event automatically by the runtime
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
        // disable printing the name of the module in every log line.
        .with_target(false)
        // disabling time is handy because CloudWatch will add the ingestion time.
        .without_time()
        .init();

    run(service_fn(function_handler)).await
}
