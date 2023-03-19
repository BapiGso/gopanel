// 获取Canvas元素
var cpuCanvas = document.getElementById("cpu-chart");
var memoryCanvas = document.getElementById("memory-chart");
var diskCanvas = document.getElementById("disk-chart");

// 设置数据
var cpuData = {
  labels: [
    "已用CPU",
    "未使用CPU"
  ],
  datasets: [
    {
      data: [75, 25],
      backgroundColor: [
        "#36A2EB",
        "#FF6384"
      ]
    }]
};

var memoryData = {
  labels: [
    "已用内存",
    "未使用内存"
  ],
  datasets: [
    {
      data: [60, 40],
      backgroundColor: [
        "#36A2EB",
        "#FF6384"
      ]
    }]
};

var diskData = {
  labels: [
    "已用硬盘",
    "未使用硬盘"
  ],
  datasets: [
    {
      data: [10, 90],
      backgroundColor: [
        "#36A2EB",
        "#FF6384"
      ]
    }]
};

// 设置选项
var options = {
  circumference: Math.PI,
  rotation: -Math.PI
};

// 实例化Chart
var cpuChart = new Chart(cpuCanvas, {
  type: 'doughnut',
  data: cpuData,
  options: options
});

var memoryChart = new Chart(memoryCanvas, {
  type: 'doughnut',
  data: memoryData,
  options: options
});

var diskChart = new Chart(diskCanvas, {
  type: 'doughnut',
  data: diskData,
  options: options
});