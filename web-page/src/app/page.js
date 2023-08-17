"use client"


import { useSession } from 'next-auth/react'
import styles from './page.module.css'
export default function Home() {

  const {data, status} = useSession()
  console.log(data, "session")
  
  return (
    <div>
      <h1>This is Torrent2Video Index Page</h1>
      {status === "authenticated" && 
      <div className={styles.inputContainer}>
        <input type="text" placeholder="Torrent" className={styles.input} required/>
        <button type="submit" className={styles.button}>Now</button>
      </div>
      }
    </div>
  )
}
