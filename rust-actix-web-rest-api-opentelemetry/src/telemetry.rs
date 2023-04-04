use actix_web_opentelemetry::{PrometheusMetricsHandler, RequestMetricsBuilder};
use dotenv::dotenv;
use opentelemetry::sdk::export::metrics::aggregation;
use opentelemetry::sdk::metrics::{controllers, processors};
use opentelemetry::{global, sdk};
use tracing_bunyan_formatter::{BunyanFormattingLayer, JsonStorageLayer};
use tracing_subscriber::{EnvFilter, Registry};
use tracing_subscriber::{prelude::*};

#[derive(Debug, Clone)]
pub struct OpenTelemetryStack {
    request_metrics: actix_web_opentelemetry::RequestMetrics,
    metrics_handler: PrometheusMetricsHandler,
}

impl OpenTelemetryStack {
    pub fn new() -> Self {
        dotenv().ok();
        let app_name = std::env::var("CARGO_BIN_NAME").unwrap_or("demo".to_string());

        global::set_text_map_propagator(opentelemetry_jaeger::Propagator::new());
        let tracer = opentelemetry_jaeger::new_agent_pipeline()
            .with_endpoint(std::env::var("JAEGER_ENDPOINT").unwrap_or("localhost:6831".to_string()))
            .with_service_name(app_name.clone())
            .install_batch(opentelemetry::runtime::Tokio)
            .expect("Failed to install OpenTelemetry tracer.");

        let telemetry = tracing_opentelemetry::layer().with_tracer(tracer);
        let env_filter = EnvFilter::try_from_default_env().unwrap_or(EnvFilter::new("INFO"));
        let formatting_layer = BunyanFormattingLayer::new(app_name.clone().into(), std::io::stdout);
        let subscriber = Registry::default()
            .with(telemetry)
            .with(JsonStorageLayer)
            .with(formatting_layer)
            .with(env_filter);
        tracing::subscriber::set_global_default(subscriber)
            .expect("Failed to install `tracing` subscriber.");

        let controller = controllers::basic(processors::factory(
            sdk::metrics::selectors::simple::histogram([0.1, 0.5, 1.0, 2.0, 5.0, 10.0]),
            aggregation::cumulative_temporality_selector(),
        )).build();
        let prometheus_exporter = opentelemetry_prometheus::exporter(controller).init();
        let meter = global::meter("global");
        let request_metrics = RequestMetricsBuilder::new().build(meter);
        let metrics_handler = PrometheusMetricsHandler::new(prometheus_exporter.clone());
        Self {
            request_metrics,
            metrics_handler
        }
    }

    pub fn metrics(&self) -> actix_web_opentelemetry::RequestMetrics {
        self.request_metrics.clone()
    }

    pub fn metrics_handler(&self) -> PrometheusMetricsHandler {
        self.metrics_handler.clone()
    }
}
