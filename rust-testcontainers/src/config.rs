pub struct Config {
    pub mongodb_uri: String,
}

impl Config {
    pub fn new() -> Self {
        dotenvy::dotenv().ok();
        let uri = dotenvy::var("MONGODB_URI").unwrap();
        Self {
            mongodb_uri: uri,
        }
    }
    pub fn new_mongodb_uri(mongodb_uri: String) -> Self {
        Self {
            mongodb_uri
        }
    }
}
