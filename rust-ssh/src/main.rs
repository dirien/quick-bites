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
