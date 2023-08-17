"use client"
import {useSession, signIn } from "next-auth/react";
import styles from './page.module.css'
import { useRouter } from "next/navigation";
import Link from "next/link";

function Register(){
    const router = useRouter()
    const {data: session, status} = useSession();

    if (status === "loading") return <p>Loading...</p>
    if (status === "authenticated") router.push("/")

    const handleSubmit = async (e) =>{
        e.preventDefault();
        const username = e.target[0].value;
        const password = e.target[1].value;
        const password_confirm = e.target[2].value;
        try {
            const res = await fetch('/goapi/v1/user/register', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ username, password, password_confirm }),
            })
            console.log(res.json())
        } catch (error) {
            console.log(error)
        }
    }

    return (
        <div className={styles.container}>
            <form className={styles.form} onSubmit={handleSubmit}>
                <input type="text" placeholder="Username" className={styles.input} required/>
                <input type="password" placeholder="Password" className={styles.input} required/>
                <input type="password" placeholder="PasswordConfirm" className={styles.input} required/>
                <button type="submit" className={styles.button}>Register</button>
            </form>
            <Link href="/login"/>
        </div>
    )
}

export default Register;