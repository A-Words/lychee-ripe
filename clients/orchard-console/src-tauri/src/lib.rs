use base64::engine::general_purpose::URL_SAFE_NO_PAD;
use base64::Engine;
use rand::RngCore;
use reqwest::blocking::Client;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::io::{BufRead, BufReader, Write};
use std::net::TcpListener;
use std::process::Command;
use std::thread;
use std::time::{Duration, Instant};
use url::Url;

#[derive(Deserialize)]
struct DiscoveryDocument {
  authorization_endpoint: String,
  token_endpoint: String,
}

#[derive(Deserialize, Serialize)]
struct TokenResponse {
  access_token: String,
  #[serde(default)]
  id_token: Option<String>,
  #[serde(default)]
  expires_in: Option<u64>,
}

#[tauri::command]
fn run_oidc_loopback_login(issuer_url: String, client_id: String, scope: String) -> Result<TokenResponse, String> {
  let issuer = issuer_url.trim().trim_end_matches('/').to_string();
  let client_id = client_id.trim().to_string();
  let scope = if scope.trim().is_empty() {
    "openid profile email".to_string()
  } else {
    scope.trim().to_string()
  };

  if issuer.is_empty() || client_id.is_empty() {
    return Err("missing oidc configuration".into());
  }

  let client = Client::builder()
    .timeout(Duration::from_secs(15))
    .build()
    .map_err(|err| err.to_string())?;

  let discovery_url = format!("{issuer}/.well-known/openid-configuration");
  let discovery = client
    .get(discovery_url)
    .send()
    .and_then(|resp| resp.error_for_status())
    .map_err(|err| err.to_string())?
    .json::<DiscoveryDocument>()
    .map_err(|err| err.to_string())?;

  let listener = TcpListener::bind("127.0.0.1:0").map_err(|err| err.to_string())?;
  listener.set_nonblocking(true).map_err(|err| err.to_string())?;
  let port = listener.local_addr().map_err(|err| err.to_string())?.port();
  let redirect_uri = format!("http://127.0.0.1:{port}/callback");

  let state = random_string(32);
  let code_verifier = random_string(64);
  let code_challenge = pkce_challenge(&code_verifier);

  let mut authorize_url = Url::parse(&discovery.authorization_endpoint).map_err(|err| err.to_string())?;
  authorize_url
    .query_pairs_mut()
    .append_pair("client_id", &client_id)
    .append_pair("response_type", "code")
    .append_pair("scope", &scope)
    .append_pair("redirect_uri", &redirect_uri)
    .append_pair("state", &state)
    .append_pair("code_challenge", &code_challenge)
    .append_pair("code_challenge_method", "S256");

  open_system_browser(authorize_url.as_str())?;

  let callback = wait_for_callback(&listener, &state)?;
  client
    .post(&discovery.token_endpoint)
    .form(&[
      ("grant_type", "authorization_code"),
      ("client_id", client_id.as_str()),
      ("code", callback.code.as_str()),
      ("code_verifier", code_verifier.as_str()),
      ("redirect_uri", redirect_uri.as_str()),
    ])
    .send()
    .and_then(|resp| resp.error_for_status())
    .map_err(|err| err.to_string())?
    .json::<TokenResponse>()
    .map_err(|err| err.to_string())
}

struct CallbackPayload {
  code: String,
}

fn wait_for_callback(listener: &TcpListener, expected_state: &str) -> Result<CallbackPayload, String> {
  let start = Instant::now();
  while start.elapsed() < Duration::from_secs(120) {
    match listener.accept() {
      Ok((mut stream, _)) => {
        let mut request_line = String::new();
        let mut reader = BufReader::new(&mut stream);
        reader.read_line(&mut request_line).map_err(|err| err.to_string())?;
        let path = request_line
          .split_whitespace()
          .nth(1)
          .ok_or_else(|| "invalid callback request".to_string())?;
        let url = Url::parse(&format!("http://localhost{path}")).map_err(|err| err.to_string())?;
        let code = url
          .query_pairs()
          .find(|(key, _)| key == "code")
          .map(|(_, value)| value.to_string())
          .ok_or_else(|| "missing authorization code".to_string())?;
        let state = url
          .query_pairs()
          .find(|(key, _)| key == "state")
          .map(|(_, value)| value.to_string())
          .ok_or_else(|| "missing callback state".to_string())?;
        if state != expected_state {
          return Err("callback state mismatch".into());
        }

        let response = "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n\r\n<html><body><h1>Login complete</h1><p>You can return to Lychee Ripe.</p></body></html>";
        stream.write_all(response.as_bytes()).map_err(|err| err.to_string())?;
        stream.flush().map_err(|err| err.to_string())?;
        return Ok(CallbackPayload { code });
      }
      Err(err) if err.kind() == std::io::ErrorKind::WouldBlock => {
        thread::sleep(Duration::from_millis(100));
      }
      Err(err) => return Err(err.to_string()),
    }
  }

  Err("oidc callback timed out".into())
}

fn open_system_browser(url: &str) -> Result<(), String> {
  #[cfg(target_os = "windows")]
  {
    Command::new("cmd")
      .args(["/C", "start", "", url])
      .spawn()
      .map_err(|err| err.to_string())?;
    return Ok(());
  }

  #[cfg(target_os = "macos")]
  {
    Command::new("open").arg(url).spawn().map_err(|err| err.to_string())?;
    return Ok(());
  }

  #[cfg(all(unix, not(target_os = "macos")))]
  {
    Command::new("xdg-open")
      .arg(url)
      .spawn()
      .map_err(|err| err.to_string())?;
    return Ok(());
  }
}

fn random_string(length: usize) -> String {
  let mut bytes = vec![0u8; length];
  rand::thread_rng().fill_bytes(&mut bytes);
  URL_SAFE_NO_PAD.encode(bytes)
}

fn pkce_challenge(verifier: &str) -> String {
  let digest = Sha256::digest(verifier.as_bytes());
  URL_SAFE_NO_PAD.encode(digest)
}

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
  tauri::Builder::default()
    .setup(|app| {
      if cfg!(debug_assertions) {
        app.handle().plugin(
          tauri_plugin_log::Builder::default()
            .level(log::LevelFilter::Info)
            .build(),
        )?;
      }
      Ok(())
    })
    .invoke_handler(tauri::generate_handler![run_oidc_loopback_login])
    .run(tauri::generate_context!())
    .expect("error while running tauri application");
}
