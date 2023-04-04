use actix_web::{get, web, App, HttpResponse, HttpServer, Responder, Result, Handler, HttpRequest};
use actix_web_opentelemetry::RequestTracing;
use opentelemetry::{Context, global};
use serde::Serialize;
use crate::repository::database::Database;

mod api;
mod models;
mod repository;
mod telemetry;

#[derive(Serialize)]
pub struct Response {
    pub message: String,
}

#[get("/health")]
async fn healthcheck() -> impl Responder {
    let response = Response {
        message: "Everything is working fine".to_string(),
    };
    HttpResponse::Ok().json(response)
}

#[get("/metrics")]
async fn metrics(telemetry: web::Data<telemetry::OpenTelemetryStack>, db: web::Data<Database>, request: HttpRequest) -> impl Responder {
    let categories = db.get_categories();
    let todos = db.get_todos();

    let meter = global::meter("global");
    let todo_count = meter.i64_observable_gauge("todo_count").with_description("Number of todos").init();
    let category_count = meter.i64_observable_gauge("category_count").with_description("Number of categories").init();

    let cx = Context::current();
    todo_count.observe(&cx, todos.len() as i64, &[]);
    category_count.observe(&cx, categories.len() as i64, &[]);
    telemetry.metrics_handler().call(request).await
}

async fn not_found() -> Result<HttpResponse> {
    let response = Response {
        message: "Resource not found".to_string(),
    };
    Ok(HttpResponse::NotFound().json(response))
}

#[actix_web::main]
async fn main() -> std::io::Result<()> {
    let todo_db = repository::database::Database::new();
    let app_data = web::Data::new(todo_db);

    let telemetry = telemetry::OpenTelemetryStack::new();
    let telemetry_data = web::Data::new(telemetry.clone());

    HttpServer::new(move || {
        App::new()
            .app_data(app_data.clone())
            .app_data(telemetry_data.clone())
            .configure(api::api::config)
            .service(healthcheck)
            .service(metrics)
            .default_service(web::route().to(not_found))
            .wrap(actix_web::middleware::Logger::default())
            .wrap(RequestTracing::new())
            .wrap(telemetry.metrics())
    })
        .bind(("127.0.0.1", 8080))?
        .run()
        .await
}
