<template>
    <div class="matches-container">
      <h1 class="title">Matches</h1>
      <div v-for="match in matches" :key="match.ID" class="matches">
        <div class="match-card" @click="toggleMatch(match.ID)">
          <span class="match-date">{{ match.Date }}</span>
          <table class="match-score">
            <thead>
              <tr>
                <th v-for="team in match.Teams" :key="team.ID">{{ team.Colour }}</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td v-for="team in match.Teams" :key="team.ID">{{ team.Score }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-if="expandedMatch === match.ID">
          <table class="match-detail">
            <thead>
              <tr>
                <th>Goal Number</th>
                <th v-for="team in match.Teams" :key="team.ID">{{ team.Colour }}</th>
                <th>Goal Number</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rowIndex in Math.max(...match.Teams.map(team => team.Players.length))" :key="rowIndex">
                <td>{{ match.Teams[0].Players[rowIndex - 1]?.GoalNumber }}</td>
                <td v-for="team in match.Teams" :key="team.ID">{{ team.Players[rowIndex - 1]?.Name }}</td>
                <td>{{ match.Teams[1].Players[rowIndex - 1]?.GoalNumber }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </template>  
  
  <script>
  import { getMatchesDetails } from '@/services/api';

  
  export default {
    data() {
    return {
      matches: [],
      expandedMatch: null, // Tracks which match is expanded
    };
  },
  methods: {
    toggleMatch(matchID) {
      // Toggle the expanded match ID
      this.expandedMatch = this.expandedMatch === matchID ? null : matchID;
    },
  },
    async created() {
      try {
        const matches = await getMatchesDetails();

        console.log("matches", matches);
        // Validate matches data
        if (!Array.isArray(matches)) {
            throw new Error('Invalid matches data');
        }  
        matches.forEach(match => {
          if (!match.ID || !match.Date || !Array.isArray(match.Teams)) {
            throw new Error('Invalid match structure');
          }
          match.Teams.forEach(team => {
            if (!team.ID || !team.Colour || !Array.isArray(team.Players)) {
              throw new Error('Invalid team structure');
            }
          });
        });

        this.matches = matches;
      } catch (error) {
        console.error('Error fetching matches:', error);
      }
    },
  };
  </script>
  
  <style>
  .matches-container {
    margin-left: 250px;
    flex: 1;
    padding: 20px;
    background-color: #f8f8f8;
    border-radius: 10px;
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
    width: calc(100% - 250px); /* Prend en compte la largeur de la sidebar */
    min-height: 100vh;
    box-sizing: border-box; /* Inclut le padding dans la largeur */
  }
  
  .title {
    color: #4CAF50;
    font-size: 28px;
    font-weight: bold;
    text-align: center;
    margin-bottom: 20px;
  }
  
  .matches {
    margin-bottom: 20px;
    border: 1px solid #ddd;
    border-radius: 8px;
    background-color: white;
    overflow: hidden;
    transition: transform 0.3s ease, box-shadow 0.3s ease;
  }
  
  .matches:hover {
    transform: scale(1.02);
    box-shadow: 0 6px 12px rgba(0, 0, 0, 0.15);
  }
  
  .match-card {
    padding: 15px;
    cursor: pointer;
    background-color: #4CAF50;
    color: white;
    border-bottom: 1px solid #ddd;
    transition: background-color 0.3s ease;
  }
  
  .match-card:hover {
    background-color: #3e8e41;
  }
  
  .match-date {
    font-size: 16px;
    font-weight: bold;
    color: white;
  }
  
  .match-score {
    width: 100%;
    border-collapse: collapse;
    margin-top: 10px;
  }
  
  .match-score th {
    background-color: #ddd;
    color: #333;
    padding: 10px;
    text-align: center;
  }
  
  .match-score td {
    border: 1px solid #ddd;
    color: #4CAF50;
    padding: 10px;
    text-align: center;
  }
  
  .match-score tbody tr:nth-child(even) {
    background-color: #f9f9f9;
  }
  
  .match-score tbody tr:nth-child(odd) {
    background-color: white;
  }
  
  .match-detail {
    width: 100%;
    border-collapse: collapse;
    margin-top: 15px;
    background-color: #f0f0f0;
    border-radius: 8px;
  }
  
  .match-detail th {
    background-color: #ddd;
    color: #333;
    padding: 10px;
    text-align: center;
  }
  
  .match-detail td {
    border: 1px solid #ddd;
    padding: 10px;
    text-align: center;
  }
  
  .match-detail tbody tr:nth-child(even) {
    background-color: #e9e9e9;
  }
  
  .match-detail tbody tr:nth-child(odd) {
    background-color: white;
  }
  </style>
  