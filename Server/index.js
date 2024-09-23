const express=require('express');
const dotenv=require('dotenv');
dotenv.config();
const makeConnection=require('./config/mongoClient')
const app=express();
const locationRoute=require('./routes/locationRoute')
const {pool,listTables}=require('./config/postgresClient');
// makePsConnect();
makeConnection();
listTables();


// note that the library takes care of URI encoding
// opencage
//   .geocode({ q: 'PSG college of Technology, 625218' })
//   .then((data) => {
//     // console.log(JSON.stringify(data));
//     if (data.status.code === 200 && data.results.length > 0) {
//       const place = data.results[0];
//       console.log(place.formatted);
//       console.log(place.geometry);
//       console.log(place.annotations.timezone.name);
//     } else {
//       console.log('Status', data.status.message);
//       console.log('total_results', data.total_results);
//     }
//   })
//   .catch((error) => {
//     // console.log(JSON.stringify(error));
//     console.log('Error', error.message);
//     // other possible response codes:
//     // https://opencagedata.com/api#codes
//     if (error.status.code === 402) {
//       console.log('hit free trial daily limit');
//       console.log('become a customer: https://opencagedata.com/pricing');
//     }
//   });

app.use("/attraction",locationRoute);
  
  


const port=process.env.PORT||3001;
app.listen(port,()=>{
    console.log(`The server is listening on port ${port}`);
    
})

