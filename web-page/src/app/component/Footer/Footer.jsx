import React from "react";
import styles from './Footer.module.css'
import Image from "next/image";

const Footer = () =>{
    return (
        <div className={styles.container}>
            <div>haojiahuo haojiahuo </div>
            <div>
                <div className={styles.imgContainer}>
                    <Image fill={true} src={"/vercel.svg"} alt={"shanghai lujiazui"}/>
                </div>
            </div>
        </div>
    )
}

export default Footer