"use client"
import React from "react";
import styles from './page.module.css'
import Link from "next/link";

const Register = () =>{
    return (
        <div className={styles.container}>
            <form className={styles.form}>
                <input type="text" placeholder="Username" className={styles.input} required/>
                <input type="email" placeholder="Email" className={styles.input} required/>
                <input type="password" placeholder="Password" className={styles.input} required/>
                <button type="submit" className={styles.button}>Register</button>
            </form>
            <Link href="/dashboard/login">Login</Link>
        </div>
    )
}

export default Register;