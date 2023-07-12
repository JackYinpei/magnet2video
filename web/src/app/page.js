'use client'
import styles from './page.module.css'
import React, { useState, useEffect } from 'react'

export default function Home() {
  const token = localStorage.getItem("token")
  const user = localStorage.getItem("username")
  const islogin = token ? true : false
  // on component did mount
  useEffect(() => {
    if (!islogin) {
      return
    }
    const response = fetch('/api/v1/me', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': token,
        'User': user,
      },
      // body: JSON.stringify({ Authorization: token, user:  user}),
    }).then(response => response.json())
    .then(data => {
      console.log(data);
      // 处理返回的数据
    })
    .catch(error => {
      console.error('Error:', error);
    });
  }, [])

  return (
    <main className={styles.main}>
      <div className={styles.description}>
        <h1 className={styles.title}>haojiahuo</h1>
      </div>
    </main>
  )
}
