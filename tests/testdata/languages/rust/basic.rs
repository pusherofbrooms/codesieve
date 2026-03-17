use std::collections::HashMap;

pub struct AuthService {
    token: String,
}

impl AuthService {
    pub fn login(&self, user: &str) -> bool {
        !user.is_empty()
    }
}

pub fn build_index() -> HashMap<String, usize> {
    HashMap::new()
}
