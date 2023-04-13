<template>
  <div>
    <h2>User Info</h2>
    <div>
      <p>Email: {{ email }}</p>
      <p>Name: {{ name }}</p>
    </div>
  </div>
</template>

<script>
export default {
  data() {
    return {
      email: '',
      name: ''
    };
  },
  async created() {
    const jwt = localStorage.getItem('jwt');

    if (!jwt) {
      this.$router.push('/login');
      return;
    }

    const response = await fetch('/api/userinfo', {
      headers: {
        Authorization: `Bearer ${jwt}`
      }
    });

    if (response.ok) {
      const data = await response.json();
      this.email = data.email;
      this.name = data.name;
    } else {
      this.$router.push('/login');
    }
  }
};
</script>
