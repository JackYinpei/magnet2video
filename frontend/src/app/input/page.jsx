'use client'
import React, {useState} from "react"
import styles from './page.module.css'
import ScriptLoad from "../component/ScriptLoad/ScriptLoad.jsx"
import Script from "next/script"

const Input = () => {

    const [client, setClient] = useState(undefined)

    const createClient = () => {
        // 直接调用 WebTorrent 暴露的函数
        console.log("load client")
        setClient(new WebTorrent())

        // const torrentId = 'magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent'

        // see tutorials.md for a full example of streaming media using service workers
        // navigator.serviceWorker.register('sw.min.js')
        // const controller = navigator.serviceWorker.ready
        // client.createServer({ controller })

        // client.add(torrentId, torrent => {
        // Torrents can contain many files. Let's use the .mp4 file
        // const file = torrent.files.find(file => {
        //     return file.name.endsWith('.mp4')
        // })
        // console.log(file, "files")
        // })
    }

    const getFiles = () => {
        const magnet = document.querySelector('input').value
        if (!magnet) {
            console.log("no magnet")
            return
        }
        console.log(magnet, "get magnet")
        client.add(magnet, torrent => {
            // Torrents can contain many files. Let's use the .mp4 file
            const file = torrent.files.find(file => {
                return file.name.endsWith('.mp4')
            })
            console.log(file, "files")
        })
    }
    return (
        <ScriptLoad>
            <div className={styles.container}>
                <div className={styles.inputcontainer}>
                    <input className={styles.input} type="text" placeholder="magnet" />
                    <button className={styles.button} onClick={getFiles}>GetFilesInstantly</button>
                </div>
            </div>
            <Script src="https://cdn.jsdelivr.net/npm/webtorrent/webtorrent.min.js" strategy="afterInteractive" onLoad={createClient}/>
        </ScriptLoad>
    )
}

export default Input