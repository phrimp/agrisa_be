const statusElement = document.getElementById('status');

// Function to request permission
function requestNotificationPermission() {
  if ('Notification' in window) {
    if (Notification.permission === 'default') {
      Notification.requestPermission().then((permission) => {
        statusElement!.textContent = `Permission: ${permission}`;
        if (permission === 'granted') {
          new Notification('Permission granted!', {
            body: 'You can now receive notifications.',
            icon: '/favicon.ico',
          });
        }
      });
    } else {
      statusElement!.textContent = `Current permission: ${Notification.permission}`;
    }
  } else {
    statusElement!.textContent = 'Notifications not supported in this browser.';
  }
}

// Request permission on page load
window.addEventListener('load', requestNotificationPermission);
