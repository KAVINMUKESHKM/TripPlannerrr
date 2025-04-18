const express = require('express');
const userController = require('../controllers/userController');
const verifyToken = require('../middleware/verifyToken');
const router = express.Router();

router.post('/register', userController.registerUser);


router.post('/login', userController.loginUser);


// router.get('/:id',verifyToken, userController.getUserProfile);


// router.put('/:id',verifyToken, userController.updateUserProfile);

router.post('/logout',verifyToken,userController.logout);

module.exports = router;
