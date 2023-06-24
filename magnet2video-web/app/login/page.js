"use client";
// import styles from "./styles.module.css";
// import { useRef } from 'react';


// export default function Login() {
//   const usernameRef = useRef();
//   const passwordRef = useRef();
//   function handleSubmit(e) {
//     e.preventDefault();
//     const username = usernameRef.current.value;
//     const password = passwordRef.current.value;
//     const data = {
//       username,
//       password,
//     };
//     fetch("/api/v1/user/login", {
//       method: "POST",
//       headers: {
//         "Content-Type": "application/json",
//       },
//       body: JSON.stringify(data),
//     }).then((response) => {
//       if (response.status === 200) {
//         // save jwt token in local storage
//         console.log(response, "response")
//         localStorage.setItem("token", response.headers.get("Authorization"));
//         let token = localStorage.getItem("token");
//         console.log(token);
//       } else {
//         // get jwt token in local storage
//         let token = localStorage.getItem("token");
//         // get jwt token in local storage
        
//         console.log("Login failed", token);
//       }
//     });
//   }
//   return (
//     <div className={styles.container}>
//       <h1 className={styles.title}>Login</h1>
//       <grid className={styles.logincard}>
//         <div className={styles.logincarditem}>
//           <form onSubmit={handleSubmit}>
//             <div>
//               <label htmlFor="username">Username</label>
//               <input
//                 type="text"
//                 required
//                 id="username"
//                 ref={usernameRef}
//               ></input>
//             </div>
//             <div>
//               <label htmlFor="password">Password</label>
//               <input type="password" id="password" ref={passwordRef}></input>
//             </div>
//             <div>
//               <label>remeber me</label>
//               <button>Login</button>
//             </div>
//           </form>
//         </div>
//       </grid>
//     </div>
//   );
// }


import * as React from 'react';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import CssBaseline from '@mui/material/CssBaseline';
import TextField from '@mui/material/TextField';
import FormControlLabel from '@mui/material/FormControlLabel';
import Checkbox from '@mui/material/Checkbox';
import Link from '@mui/material/Link';
import Grid from '@mui/material/Grid';
import Box from '@mui/material/Box';
import LockOutlinedIcon from '@mui/icons-material/LockOutlined';
import Typography from '@mui/material/Typography';
import Container from '@mui/material/Container';
import { createTheme, ThemeProvider } from '@mui/material/styles';

function Copyright(props) {
  return (
    <Typography variant="body2" color="text.secondary" align="center" {...props}>
      {'Copyright © '}
      <Link color="inherit" href="https://mui.com/">
        Your Website
      </Link>{' '}
      {new Date().getFullYear()}
      {'.'}
    </Typography>
  );
}

// TODO remove, this demo shouldn't need to reset the theme.

const defaultTheme = createTheme();

export default function SignIn() {
  const handleSubmit = (event) => {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    console.log({
      email: data.get('email'),
      password: data.get('password'),
    });
  };

  return (
    <ThemeProvider theme={defaultTheme}>
      <Container component="main" maxWidth="xs">
        <CssBaseline />
        <Box
          sx={{
            marginTop: 8,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
          }}
        >
          <Avatar sx={{ m: 1, bgcolor: 'secondary.main' }}>
            <LockOutlinedIcon />
          </Avatar>
          <Typography component="h1" variant="h5">
            Sign in
          </Typography>
          <Box component="form" onSubmit={handleSubmit} noValidate sx={{ mt: 1 }}>
            <TextField
              margin="normal"
              required
              fullWidth
              id="email"
              label="Email Address"
              name="email"
              autoComplete="email"
              autoFocus
            />
            <TextField
              margin="normal"
              required
              fullWidth
              name="password"
              label="Password"
              type="password"
              id="password"
              autoComplete="current-password"
            />
            <FormControlLabel
              control={<Checkbox value="remember" color="primary" />}
              label="Remember me"
            />
            <Button
              type="submit"
              fullWidth
              variant="contained"
              sx={{ mt: 3, mb: 2 }}
            >
              Sign In
            </Button>
            <Grid container>
              <Grid item xs>
                <Link href="#" variant="body2">
                  Forgot password?
                </Link>
              </Grid>
              <Grid item>
                <Link href="#" variant="body2">
                  {"Don't have an account? Sign Up"}
                </Link>
              </Grid>
            </Grid>
          </Box>
        </Box>
        <Copyright sx={{ mt: 8, mb: 4 }} />
      </Container>
    </ThemeProvider>
  );
}