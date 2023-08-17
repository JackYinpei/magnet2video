"use client"
import React from "react";
import Link from "next/link";
import styles from './navbar.module.css'
import DarkmodeToggle from "../DarkmodeToggle/DarkmodeToggle";
import {signOut, useSession} from "next-auth/react";
import {useRouter} from "next/navigation";

const  Links = [
    {
        id:1,
        title:"Home",
        url:"/",
    },
    {
        id:2,
        title:"Blog",
        url:"/blog",
    },
    {
        id:3,
        title:"Dashboard",
        url:"/dashboard",
    },
];

function Navbar(){

    const router = useRouter()
    const session = useSession()

    return (
        <div className={styles.container}>
            <Link href={"/"} className={styles.logo}>Magnet2Video</Link>
            <div className={styles.links}>
                <DarkmodeToggle/>
                {Links.map(link=>(
                    <Link href={link.url} key={link.id} className={styles.link}>
                        {link.title}
                    </Link>
                ))}
                {
                    session.status === "authenticated" && <button
                        className={styles.logout}
                        onClick={signOut}>Logout</button>
                }
                {
                    session.status === "unauthenticated" && <button
                    className={styles.logout}
                    onClick={()=>{router.push("/login")}}>Login</button>
                }
            </div>
        </div>
    )
}

export default Navbar