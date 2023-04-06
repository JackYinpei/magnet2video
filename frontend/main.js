// Define a function to perform the login process
function login() {
  // Get the username and password from the login form
  var username = document.getElementById("username").value;
  var password = document.getElementById("password").value;

  // Send a POST request to the server to authenticate the user
  fetch("/api/v1/user/login", {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({username: username, password: password})
  })
  .then(response => {
    if (response.ok) {
      // If the login is successful, save the JWT token in localStorage
      response.json().then(data => {
        localStorage.setItem("jwt", data.token);
        // Redirect to the user info page
        window.location.href = "/user";
      });
    } else {
      // If the login fails, display an error message
      alert("Login failed. Please check your username and password.");
    }
  })
  .catch(error => {
    console.error(error);
    alert("An error occurred while processing your request.");
  });

  // Send a GET request to the server to retrieve user info
fetch("/user", {
  headers: {
    "Authorization": "Bearer " + localStorage.getItem("jwt")
  }
})
.then(response => {
  if (response.ok) {
    // If the request is successful, display the user info
    response.json().then(data => {
      // TODO: display user info in the UI
    });
  } else {
    // If the request fails, display an error message
    alert("An error occurred while retrieving user info.");
  }
})
.catch(error => {
  console.error(error);
  alert("An error occurred while processing your request.");
});

}
