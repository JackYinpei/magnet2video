'use client'
import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import style from './page.module.css'
import { Card, Grid, Text, Button, Row, Input, Spacer } from "@nextui-org/react";


const Login = () => {

  const router = useRouter();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  const handleUsernameChange = (event) => {
    setUsername(event.target.value);
  };

  const handlePasswordChange = (event) => {
    setPassword(event.target.value);
  };

  const handleSubmit = async () => {

    // 发送用户名和密码到后端
    const response = await fetch('/api/v1/user/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ username, password }),
    });

    if (response.ok) {
      // 登录成功，获取令牌
      const data = await response.json();
      const token = data.data.Authorization
      const username = data.data.username
      localStorage.setItem("token", token)
      localStorage.setItem("username", username)
      router.push('/');
    } else {
      // 登录失败，显示错误消息
      const data = await response.json();
      const { message } = data;
      console.error('Login failed:', message);
    }
  };

  return (
    <div className={style.container}>
      <div className={style.left} style={{ backgroundImage: `url("/beach.jpeg")` }}>
      </div>
      <div className={style.right}>
        <Grid.Container gap={2} justify="center">
          <Grid sm={12} md={5}>
            <Card css={{ mw: "330px" }}>
              <Card.Header>
                <Text b>Login</Text>
              </Card.Header>
              <Card.Divider />
              <Card.Body css={{ py: "$10" }}>
                <Input labelPlaceholder="Username" onChange={handleUsernameChange}/>
                <Spacer y={1} />
                <Input.Password
                  labelPlaceholder="Password" onChange={handlePasswordChange}/>
              </Card.Body>
              <Card.Divider />
              <Card.Footer>
                <Row justify="flex-end">
                  <Button size="sm" light onPress={handleSubmit}>
                    Login
                  </Button>
                  <Button size="sm">Signup</Button>
                </Row>
              </Card.Footer>
            </Card>
          </Grid>
        </Grid.Container>
      </div>
      <style jsx>{`
        .container {
          display: flex;
          height: 100vh;
        }
        .left {
          flex: 1;
          display: flex;
          justify-content: center;
          align-items: center;
        }
        .right {
          flex: 1;
          display: flex;
          justify-content: center;
          align-items: center;
        }
        img {
          max-width: 100%;
          max-height: 100%;
        }
        form {
          display: flex;
          flex-direction: column;
          align-items: center;
        }
        input, button {
          margin-bottom: 10px;
        }
        left-img {
          object-fit: cover; /* 使用cover值实现图片铺满并裁剪 */
          // width: 100%;
          height: 100%;
        }
      `}</style>
    </div>
  );
};

export default Login;
