const express = require('express');
const router = express.Router();
const taskController = require('../controllers/taskController');

router.get('/', taskController.getAllTasks);
router.get('/:id/history', taskController.getTaskHistory);
router.post('/', taskController.createTask);
router.put('/:id', taskController.updateTask);

module.exports = router;
