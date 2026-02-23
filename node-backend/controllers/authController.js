const config = require("../config");
const { validateCredentials } = require("../middleware/auth");

function login(req, res) {
  const { username, password } = req.body || {};

  if (typeof username !== "string" || typeof password !== "string") {
    res.status(400).json({
      error: "username and password are required.",
    });
    return;
  }

  if (!validateCredentials(username, password)) {
    res.status(401).json({ error: "Invalid username or password." });
    return;
  }

  res.json({
    username: config.AUTH_USERNAME,
    tokenType: "ApiKey",
    apiKey: config.AUTH_API_KEY,
  });
}

module.exports = {
  login,
};
