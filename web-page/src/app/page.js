"use client"

import { useState } from 'react'
import { useSession } from 'next-auth/react'
import styles from './page.module.css'
export default function Home() {

  async function getFiles(token, magnet) {
    const res = await fetch("/goapi/v1/magnet", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "Authorization": token,
      },
      body: JSON.stringify({magnet: magnet}),
    })

    if (!res.ok) {
      throw new Error(`Failed to fetch: ${res.status}`)
    }
    return res.json()
  }

  const {data, status} = useSession()
  console.log(data, status, "data, status")
  const [inputValue, setInputValue] = useState('')
  const [respjson, setRespjson] = useState({})

  const handleSubmit = async (e) => {
    e.preventDefault()
    console.log(inputValue, "鼠标点击")
    const magnet = inputValue
    const token = data.user.email
    const resp = await getFiles(token, magnet)
    console.log(respjson, "respjson")
  }

  
  return (
    <div>
      <h1>This is Torrent2Video Index Page</h1>
      {status === "authenticated" && 
      <div className={styles.inputContainer}>
        <input type="text" placeholder="Torrent" className={styles.input} required onChange={(e)=>{setInputValue(e.target.value)}}/>
        <button type="submit" className={styles.button} onClick={handleSubmit} >Now</button>
      </div>
      }
    </div>
  )
}
