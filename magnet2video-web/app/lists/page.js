"use client";
import React, {useEffect} from "react";
import Navbar from "../components/navbar/navbar";

export default function Lists() {
  // request and list all the files

  useEffect(() => {
    let token = localStorage.getItem("token");
    fetch("/api/v1/magnets", {
      headers: {
        'Authorization': 'bearer:' + token,
      }}).then((response) => {
        console.log(response, "response")
      })
  }, [])

  return (
    <div>
      <Navbar />
      <div className="grid-container">
        <div>
          <a>haojiahuo</a>
        </div>
        <a className="aside" />
        <div>
          <a>haojiahuo</a>
        </div>
      </div>
    </div>
  );
}
