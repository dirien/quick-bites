# Rust Development with Testcontainers

## Introduction

In this blog post, we're going to explore how to use [Testcontainers](https://testcontainers.com/) as part of our integration testing strategy in Rust. To have hands-on experience, we're going to build a simple web application that exposes a REST API to manage cars. The cars are stored in a [MongoDB](https://www.mongodb.com/) database and as a web framework, we're going to use [Actix-Web](https://actix.rs/).

%[https://testcontainers.com/] 

I highly recommend my previous blog post about the `actix-web` framework

%[https://blog.ediri.io/rust-development-creating-a-rest-api-with-actix-web-for-beginners] 

Or check my whole Rust learning journey

%[https://blog.ediri.io/series/learning-rust] 

## What is Testcontainers?

`Testcontainers` is an open-source framework to provide throwaway instances of dependencies such as databases, message brokers, or any other service that can be started in a Docker container. It's available for many programming languages such as Java, Python, Go, and Rust and allows us to write test code that allows the user to start and stop containers.

This has <mark>several advantages</mark>:

* We can run tests against real components like PostgreSQL instead of an H2 in-memory database. This allows us to use PostgreSQL-specific features like JSONB columns or full-text search.

* We can mock AWS services with Localstack. This allows us to test our code against AWS services without the need to create real resources in the cloud.

* We can run our tests also in an offline environment.

* We can also test better some edge cases like network failures or slow responses from the database.


In this blog post, we will use the Rust version of Testcontainers, which is called `rust-testcontainers`.

Testcontainers offers also preconfigured implementations called [modules](https://testcontainers.com/modules/) for different databases and services. As they are not available for Rust, I skipped them in this blog post.

%[https://testcontainers.com/modules/] 

## The Testing Pyramid  

Before we head over to the implementation, let's talk about the different testing approaches. There are three different approaches to testing strategies:

* *The Test Ice Cream Cone*

* *The Test Pyramid*

* *The Practical Test Pyramid*


### The Test Ice Cream Cone

[![https://yellow.systems/blog/choosing-the-right-automation-testing-strategy-dos-and-don-ts](https://images.ctfassets.net/0nm5vlv2ad7a/4S6PqgeIdYeIwvJBavNxIp/e621b702f425fd531e4e64d788d5a28a/choosing-the-right-automation-testing-strategy-dos-and-don-ts.png align="center")](https://yellow.systems/blog/choosing-the-right-automation-testing-strategy-dos-and-don-ts)

The Test Ice Cream Cone is a testing strategy that is often used by companies that are new to testing. The problem with this approach is that we have a lot of manual tests that are expensive to maintain and slow to execute.

The Test Ice Cream Cone is an anti-pattern and should be avoided at all costs.

### The Test Pyramid

[![](https://wpblog.semaphoreci.com/wp-content/uploads/2022/03/pyramid1.jpg align="center")](https://semaphoreci.com/blog/testing-pyramid)

The Test Pyramid is a testing strategy that was introduced by **Mike Cohn** in his book "Succeeding with Agile". The pyramid shows three levels of tests: small, medium, and large.

**<mark>Unit tests</mark>** are the primary level of the pyramid. They are small, fast, and cheap to maintain. They are focused on testing the logic in the code.

**<mark>Service tests</mark>** are the second level of the pyramid. They are medium-sized, slower, and more expensive to maintain. They are not as productive as unit tests.

And the third level of the pyramid is **<mark>UI tests</mark>**. They are large, slow, and expensive to maintain as they are very fragile since every change in the UI can break the tests.

One of the <mark>problems</mark> with the Test Pyramid is that it's not clear what service tests are which leads to the situation that developers jump directly to UI tests.

### The Practical Test Pyramid

![https://medium.com/tide-engineering-team/the-practical-test-pyramid-c4fcdbc8b497](https://miro.medium.com/v2/resize:fit:1400/1*8NtJX228Arq5fjB4LCv8jw.png align="center")

The Practical Test Pyramid is an improved version of the Test Pyramid. It was introduced by **Alister Scott** and emphasizes more on medium-level tests and manual exploratory testing.

The <mark>Services Tests</mark> from the Test Pyramid are replaced by **Component Tests**, **Integration Tests**, and **Contract Tests**.

The next improvement of the Test Pyramid is that is now more clear that manual testing is also part of the testing strategy. As we can never be 100% sure that our tests are covering all edge cases, we need to add <mark>Manual Exploratory Testing</mark> on top of our automated tests.

If you find a bug with manual testing, you should write a new test to cover this case.

## Add Testcontainers to our Project

I will not go into the details of the existing code for setting up the project. If you want to know more about this, feel free to browse the code on GitHub.

To add Testcontainers to our project, we need to add the following dependency to our project:

```bash
cargo add testcontainers
```

Now open the `main.rs` file and add following code at the end of the file:

```rust
#[cfg(test)]
mod tests {
    use std::env;
    use std::io::Read;
    use std::thread::sleep;
    use super::*;
    use actix_web::http::StatusCode;
    use actix_web::test;
    use actix_web::test::TestRequest;
    use testcontainers::{clients, Image};
    use testcontainers::core::{ExecCommand, WaitFor};
    use testcontainers::images::generic::GenericImage;
    use crate::model::CarType;
    use crate::model::Car;

    #[actix_web::test]
    async fn test_index() {
        let app = test::init_service(App::new().service(index)).await;
        let req = TestRequest::default().to_request();
        let resp = test::call_service(&app, req).await;
        assert_eq!(StatusCode::OK, resp.status());
    }

    #[actix_web::test]
    async fn test_healthcheck() {
        let app = test::init_service(App::new().service(healthcheck)).await;
        let req = TestRequest::default().uri("/health").to_request();
        let resp = test::call_service(&app, req).await;
        assert_eq!(StatusCode::OK, resp.status());
    }

    #[actix_web::test]
    async fn test_get_cars() {
        let docker = clients::Cli::default();
        let msg = WaitFor::message_on_stdout("server is ready");
        let generic = GenericImage::new("mongo", "6.0.7").with_wait_for(msg.clone())
            .with_env_var("MONGO_INITDB_DATABASE", "cars_info")
            .with_env_var("MONGO_INITDB_ROOT_USERNAME", "root")
            .with_env_var("MONGO_INITDB_ROOT_PASSWORD", "root")
            .with_exposed_port(27017);

        let node = docker.run(generic);
        let port = node.get_host_port_ipv4(27017);
        println!("Port: {}", port);

        let data = setup(Config::new_mongodb_uri(format!("mongodb://root:root@localhost:{}", port))).await;
        let app = test::init_service(App::new().app_data(data.clone()).service(get_cars).service(create_car)).await;
        let req = TestRequest::default().uri("/cars").to_request();
        let resp = test::call_service(&app, req).await;
        assert_eq!(StatusCode::OK, resp.status());
        let result: Vec<Car> = test::read_body_json(resp).await;
        assert_eq!(result.len(), 0);

        let post = create_one_test_car();
        let resp = test::call_service(&app, post.to_request()).await;
        assert_eq!(StatusCode::OK, resp.status());
        let result: Car = test::read_body_json(resp).await;
        assert_eq!(result.name, "Test");

        let req = TestRequest::default().uri("/cars").to_request();
        let resp = test::call_service(&app, req).await;
        assert_eq!(StatusCode::OK, resp.status());
        let result: Vec<Car> = test::read_body_json(resp).await;
        assert_eq!(result.len(), 1);
        assert_eq!(result[0].name, "Test");
    }

    #[actix_web::test]
    async fn test_get_car() {
        let docker = clients::Cli::default();
        let msg = WaitFor::message_on_stdout("server is ready");
        let generic = GenericImage::new("mongo", "6.0.7").with_wait_for(msg.clone())
            .with_env_var("MONGO_INITDB_DATABASE", "cars_info")
            .with_env_var("MONGO_INITDB_ROOT_USERNAME", "root")
            .with_env_var("MONGO_INITDB_ROOT_PASSWORD", "root");

        let node = docker.run(generic);
        let port = node.get_host_port_ipv4(27017);

        let data = setup(Config::new_mongodb_uri(format!("mongodb://root:root@localhost:{}", port))).await;
        let app = test::init_service(App::new().app_data(data.clone()).service(get_cars).service(create_car).service(get_car)).await;


        let create_car_req = create_one_test_car();
        let resp = test::call_service(&app, create_car_req.to_request()).await;
        assert_eq!(StatusCode::OK, resp.status());
        let new_car: CarDto = test::read_body_json(resp).await;
        assert_eq!(new_car.name, "Test");

        let get_car_req = TestRequest::get().uri(format!("/cars/{}", new_car.id.unwrap()).as_str()).to_request();
        let resp = test::call_service(&app, get_car_req).await;
        assert_eq!(StatusCode::OK, resp.status());
        let result: CarDto = test::read_body_json(resp).await;
        assert_eq!(result.name, new_car.name);
    }

    #[actix_web::test]
    async fn test_delete_car() {
        let docker = clients::Cli::default();
        let msg = WaitFor::message_on_stdout("server is ready");
        let generic = GenericImage::new("mongo", "6.0.7").with_wait_for(msg.clone())
            .with_env_var("MONGO_INITDB_DATABASE", "cars_info")
            .with_env_var("MONGO_INITDB_ROOT_USERNAME", "root")
            .with_env_var("MONGO_INITDB_ROOT_PASSWORD", "root");

        let node = docker.run(generic);
        let port = node.get_host_port_ipv4(27017);

        let data = setup(Config::new_mongodb_uri(format!("mongodb://root:root@localhost:{}", port))).await;
        let app = test::init_service(App::new().app_data(data.clone())
            .service(get_cars).service(create_car).service(get_car)
            .service(delete_car)).await;

        let create_car_req = create_one_test_car();
        let resp = test::call_service(&app, create_car_req.to_request()).await;
        assert_eq!(StatusCode::OK, resp.status());
        let new_car: CarDto = test::read_body_json(resp).await;
        assert_eq!(new_car.name, "Test");

        let new_car_id = new_car.id.unwrap();
        let get_car_req = TestRequest::get().uri(format!("/cars/{}", new_car_id).as_str()).to_request();
        let resp = test::call_service(&app, get_car_req).await;
        assert_eq!(StatusCode::OK, resp.status());
        let result: CarDto = test::read_body_json(resp).await;
        assert_eq!(result.name, new_car.name);

        let delete_car_req = TestRequest::delete().uri(format!("/cars/{}", new_car_id).as_str()).to_request();
        let resp = test::call_service(&app, delete_car_req).await;
        assert_eq!(StatusCode::OK, resp.status());

        let get_car_req = TestRequest::get().uri(format!("/cars/{}", new_car_id).as_str()).to_request();
        let resp = test::call_service(&app, get_car_req).await;
        assert_eq!(StatusCode::NOT_FOUND, resp.status());
    }

    fn create_one_test_car() -> TestRequest {
        let post = TestRequest::post().uri("/cars").set_json(&dto::CarDto {
            id: None,
            name: "Test".to_string(),
            brand: "Test".to_string(),
            year: 2021,
            r#type: CarType::Other,
        });
        post
    }
}
```

Let's describe what is happening here:

* The module defines a set of tests and imports the required dependencies.

* Each test is marked with the `#[actix_web::test]` attribute for recognition by the testing framework.

* The initial two tests (`test_index` and `test_healthcheck`) validate the response of `index` and `healthcheck` endpoints, respectively. These tests simply call the service and assert that the response is as expected.

* The `test_get_cars` function introduces the use of Testcontainers. It creates an instance of a Testcontainers client, which allows the use of Docker containers during tests. This function specifically employs MongoDB as the test container:

  * A new container is instantiated from the "`mongo`" image (version "`6.0.7`").

  * The container is prepared with specific environment variables for **MongoDB's** initial database setup.

  * An exposed port, `27017`, is specified for network communications.

  * A MongoDB URI is assembled, pointing towards the Docker-hosted MongoDB instance.

  * After the setup, test requests are sent and their responses are validated.

* The `test_get_car` and `test_delete_car` functions employ a similar testing approach as `test_get_cars`. These functions create a MongoDB container using the Testcontainers client and then execute tests that are supposed to interact with the MongoDB database.


## Testing

To run the tests, execute the following command:

```bash
cargo test
```

Because the tests are using Docker containers, the first run will take a while to download the required images. And, of course, you need Docker installed on your machine.

If everything goes well, you should see the following output:

```bash
running 5 tests
test tests::test_healthcheck ... ok
test tests::test_index ... ok
test tests::test_get_car ... ok
test tests::test_delete_car ... ok
test tests::test_get_cars ... ok

test result: ok. 5 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 6.44s
```

## Conclusion

Testcontainers undoubtedly offer a powerful framework to improve the fidelity of your application testing through the use of Docker containers. However, they are not without limitations. Here are some notable considerations based on my personal experience:

* The Rust iteration of Testcontainers is not on par with its Java or Go counterparts, resulting in the absence of certain features.

* As of July 5th, 2023, no Rust Testcontainer Modules exist, which means that you must use the GenericImage and configure the container yourself.

* Testcontainers introduce another dependency to manage in your project.

* Containers can be resource-intensive, requiring careful management to avoid overconsumption.

* Your CI/CD pipeline must not only support containers but also possess the capacity to run them efficiently.

* These requirements are equally valid if you're running tests locally on your own machine.


In conclusion, the decision to adopt Testcontainers hinges on individual project needs. I found the tool valuable and plan to use it in my non-Rust projects.

### Resources

* %[https://medium.com/tide-engineering-team/the-practical-test-pyramid-c4fcdbc8b497] 

  %[https://testcontainers.com/]
