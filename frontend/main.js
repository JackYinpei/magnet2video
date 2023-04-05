// Get DOM elements
const loginForm = document.querySelector('#login-form');
const registerForm = document.querySelector('#register-form');
const fileList = document.querySelector('#file-list');
const magnetList = document.querySelector('#magnet-list');
const fileDetails = document.querySelector('#file-details');

// Keep track of selected magnet
let selectedMagnet = null;

// Add event listeners for login and register forms
document.querySelectorAll('form').forEach((form) => {
  form.addEventListener('submit', (event) => {
    event.preventDefault();
    // TODO: Validate user input and make a server request
    // to authenticate or create user
    // For example, if authentication is successful:
    loginForm.style.display = 'none';
    registerForm.style.display = 'none';
    // and display the file list
    fileList.style.display = 'block';
    // TODO: Fetch the user's magnet list and display it in the magnet list
    // For example:
    const magnets = [
      {
        id: 1,
        name: 'Magnet 1',
        files: [
          {
            id: 1,
            name: 'File 1',
            size: 100,
            type: 'text/plain'
          },
          {
            id: 2,
            name: 'File 2',
            size: 200,
            type: 'text/plain'
          }
        ]
      },
      {
        id: 2,
        name: 'Magnet 2',
        files: [
          {
            id: 3,
            name: 'File 3',
            size: 300,
            type: 'text/plain'
          },
          {
            id: 4,
            name: 'File 4',
            size: 400,
            type: 'text/plain'
          }
        ]
      }
    ];
    for (const magnet of magnets) {
      const li = document.createElement('li');
      li.textContent = magnet.name;
      li.addEventListener('click', () => {
        // When a magnet is clicked, display its details in the file details
        // and mark it as active in the magnet list
        selectedMagnet = magnet;
        displayFileDetails();
        document.querySelectorAll('#magnet-list li').forEach((li) => {
          li.classList.remove('active');
        });
        li.classList.add('active');
      });
      magnetList.appendChild(li);
    }
  });
});

function displayFileDetails() {
  fileDetails.innerHTML = '';
  const h3 = document.createElement('h3');
  h3.textContent = selectedMagnet.name;
  fileDetails.appendChild(h3);
  const ul = document.createElement('ul');
  for (const file of selectedMagnet.files) {
    const li = document.createElement('li');
    li.innerHTML = `${file.name} (${file.size} bytes, ${file.type})`;
    ul.appendChild(li);
  }
  fileDetails.appendChild(ul);
}
