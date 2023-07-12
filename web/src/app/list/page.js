'use client'
import styles from './page.module.css'
import React, { useState, useEffect } from 'react'
import { Table } from '@nextui-org/react';

export default function Home() {
  const token = localStorage.getItem("token")
  const user = localStorage.getItem("username")
  const islogin = token ? true : false
  var items = []
  // on component did mount
  useEffect(() => {
    if (!islogin) {
      return
    }
    const response = fetch('/api/v1/magnets', {
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
      Object.keys(data.data.items).forEach(function(key){
        items.push(data.data.items[key])
      });
      // 处理返回的数据
    })
    .catch(error => {
      console.error('Error:', error);
    });
  }, [])

  return (
    <main className={styles.main}>
      <div className={styles.description}>
      {items.length !== 0 &&<Table
        aria-label="Example table with dynamic content"
        css={{
            height: "auto",
            minWidth: "100%",
        }}
        >
        <Table.Header columns={columns}>
            {(column) => (
            <Table.Column key={column.key}>{column.label}</Table.Column>
            )}
        </Table.Header>
        <Table.Body items={rows}>
            {(item) => (
            <Table.Row key={item.key}>
                {(columnKey) => <Table.Cell>{item[columnKey]}</Table.Cell>}
            </Table.Row>
            )}
        </Table.Body>
        </Table>
}
      </div>
    </main>
  )
}
