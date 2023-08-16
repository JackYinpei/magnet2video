"use client"
import React from "react";
import { signIn } from "next-auth/react";
import styles from './page.module.css'

const Login = () =>{

    const handleSubmit = (e) =>{
        e.preventDefault();
        const username = e.target[0].value;
        const email = e.target[1].value;
        const password = e.target[2].value;
        signIn("credentials", {username, email, password})
    }

    return (
        <div className={styles.container}>
            <form className={styles.form} onSubmit={handleSubmit}>
                <input type="text" placeholder="Username" className={styles.input} required/>
                <input type="email" placeholder="Email" className={styles.input} required/>
                <input type="password" placeholder="Password" className={styles.input} required/>
                <button type="submit" className={styles.button}>Login</button>
            </form>
            <button className={styles.button} onClick={() => signIn("google")}>Login with Google</button>
        </div>
    )
}

export default Login;