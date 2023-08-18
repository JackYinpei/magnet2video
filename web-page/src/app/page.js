"use client"

import { useState } from 'react'
import { useSession } from 'next-auth/react'
import styles from './page.module.css'
import axios from 'axios'

export default function Home() {

  const {data, status} = useSession()
  const [inputValue, setInputValue] = useState('')
  const [loadOk, setLoadOK] = useState(false)
  const [files, setFiles] = useState([])

  const handleSubmit = (e) => {
    e.preventDefault()
    const magnet = inputValue
    const token = data.user.email

    axios.defaults.headers.common['Authorization'] = token
    axios.defaults.headers.common['Content-Type'] = 'application/json'
    axios.post('/goapi/v1/magnet',{
      magnet: String(magnet)
    }).then((res) => {
      if (res.data.status !== 40001) {
        setLoadOK(true)
        setFiles(res.data.data.files)
        console.log(files, "files checkout is ok", res.data.data.files, "kankanzhege")
      }
    }).catch((err) => {
      console.log("返回文件失败", err)
      return err
    })
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
      {loadOk && files.length > 0 &&
        <div>
          <ol>
            {files.forEach((file) => {
              return <li>{file}</li>
            })}
          </ol>
        </div>}
    </div>
  )
}
