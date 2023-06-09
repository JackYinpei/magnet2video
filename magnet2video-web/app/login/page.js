import styles from './styles.module.css'

export default function Login(){
    return (
        <div className={styles.container}>
            <h1 className={styles.title}>Login</h1>
            <grid className={styles.logincard}>
                <div className={styles.logincarditem}>
                    <div>
                        <label>Username</label>
                        <br/>
                        <input type="text"></input>
                        <br/>
                        <label>Password</label>
                        <br/>
                        <input type="password"></input>
                        <br/>
                        <label>remeber me</label><button>Login</button>

                    </div>
                </div>
            </grid>
        </div>
    )
}