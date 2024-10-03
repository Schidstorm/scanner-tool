document.addEventListener("DOMContentLoaded", function () {
    document.body.addEventListener('onServerError', function (e) {
        const err = e.detail.value;
        const messageElement = document.getElementById('error-message');
        if (messageElement) {
            messageElement.innerText = err;
        }

        const errorElement = document.getElementById('error');
        if (errorElement) {
            errorElement.style.display = 'block';
        }
    })
});

