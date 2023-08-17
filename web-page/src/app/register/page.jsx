"use client"
import React from "react";
import { signIn } from "next-auth/react";
import styles from './page.module.css'
import { useRouter } from "next/navigation";
import Link from "next/link";

function Login(){
    const router = useRouter()

    const handleSubmit = async (e) =>{
        e.preventDefault();
        const username = e.target[0].value;
        const email = e.target[1].value;
        const password = e.target[2].value;
        try {
            const res = await fetch('/api/register', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, email, password }),
            })
            if (res.ok) {
                const { token } = await res.json()
                signIn('credentials', { username, email, password })
                router.push('/')
            }
        } catch (error) {
            console.log(error)
        }
    }

    return (
        <div className={styles.container}>
            <form className={styles.form} onSubmit={handleSubmit}>
                <input type="text" placeholder="Username" className={styles.input} required/>
                <input type="email" placeholder="Email" className={styles.input} required/>
                <input type="password" placeholder="Password" className={styles.input} required/>
                <button type="submit" className={styles.button}>Register</button>
            </form>
            <Link href="/login"/>
        </div>
    )
}

export default Login;