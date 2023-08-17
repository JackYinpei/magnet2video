"use client"
import React from "react";
import {useSession, signIn } from "next-auth/react";
import styles from './page.module.css'
import { useRouter } from "next/navigation";

function Login(){
    const router = useRouter()
    const {data: session, status} = useSession();

    if (status === "loading") return <p>Loading...</p>
    if (status === "authenticated") router.push("/")

    const handleSubmit = (e) =>{
        e.preventDefault();
        const username = e.target[0].value;
        const password = e.target[1].value;
        signIn("credentials", {username, password})

    }

    return (
        <div className={styles.container}>
            <form className={styles.form} onSubmit={handleSubmit}>
                <input type="text" placeholder="Username" className={styles.input} required/>
                <input type="password" placeholder="Password" className={styles.input} required/>
                <button type="submit" className={styles.button}>Login</button>
            </form>
            <button className={styles.button} onClick={() => signIn("google")}>Login with Google</button>
        </div>
    )
}

export default Login;