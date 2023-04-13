const loginForm = document.getElementById("login-form");

loginForm.addEventListener("submit", (event) => {
  event.preventDefault();

  const username = document.getElementById("username").value;
  const password = document.getElementById("password").value;
  alert("use javascripts")
  fetch("/api/v1/user/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
    headers: {
      "Content-Type": "application/json",
    },
  })
    .then((response) => response.json())
    .then((data) => {
      if (data.token) {
        localStorage.setItem("token", `Bearer ${data.token}`);// Store token in local storage
        const headers = new Headers();
        headers.append("Authorization", `Bearer ${data.token}`); // Add Authorization header to redirect request
        window.location.href = "/api/v1/me";
      }
    })
    .catch((error) => console.error(error));
});

function getUserInfo() {
  const token = localStorage.getItem("token");
  if (!token) {
    // handle error, e.g. redirect to login page
    return;
  }

  fetch("/api/v1/me", {
    headers: {
      "Authorization": `Bearer ${token}`
    }
  })
    .then(response => response.json())
    .then(data => console.log(data))
    .catch(error => console.error(error));
}