
<template>
  <div id="app">
    <div v-if="showImage" class="initial-view">
      <img alt="Vue logo" src="./assets/campo_image_1.jpg" @click="toggleView" style="cursor: pointer;">
      <h1 class="title-logo">Calciotto</h1>
    </div>
    <div class="app-container" v-else>
        <div class="sidebar">
            <img src="@/assets/logo.png" alt="Logo" class="logo" @click="toggleView" style="cursor: pointer;"/>
            <div class="menu">
                <button class="menu-button">Matches</button>
                <button class="menu-button">Teams</button>
                <button class="menu-button">Players</button>
            </div>
        </div>
      <MatchesAll/>
    </div>
  </div>
</template>

<script>
import MatchesAll from './components/MatchesAll.vue';

export default {
  name: 'App',
  components: {
    MatchesAll
  },
  data() {
    return {
      showImage: true
    };
  },
  methods: {
    toggleView() {
      this.showImage = !this.showImage;
    }
  }
};
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
  color: #2c3e50;
  margin-top: 60px;
}

.initial-view {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100vh;
}

.title-logo {
  color: green;
  font-size: 28px;
  font-weight: bold;
  margin-top: 20px;
}

/* Conteneur principal pour l'application */
.app-container {
  display: flex;
  min-height: 100vh;
  background-color: #f5f5f5;
}

/* Style de la sidebar */
.sidebar {
  width: 250px;
  background-color: #e0e0e0;
  padding: 20px;
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: 2px 0 5px rgba(0, 0, 0, 0.1);
  height: 100vh;
  position: fixed;
  left: 0;
  top: 0;
}

.logo {
  height: 80px;
  margin-bottom: 30px;
}

/* Style des boutons du menu */
.menu {
  width: 100%;
}

.menu-button {
  width: 100%;
  background: none;
  border: none;
  padding: 12px;
  margin-bottom: 10px;
  text-align: left;
  cursor: pointer;
  font-size: 16px;
  color: green;
  font-weight: bold;
  transition: background-color 0.3s, transform 0.3s, color 0.3s;
  border-radius: 4px;
  padding-left: 15px;
}

.menu-button:hover {
  background-color: darkgreen;
  color: white;
  transform: scale(1.02);
}
</style>
