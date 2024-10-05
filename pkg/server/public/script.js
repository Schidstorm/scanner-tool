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

    document.getElementById('result').addEventListener('click', function (e) {
        const target = e.target;
        if (closest(target, '.preview-item') && !closest(target, 'button') && !closest(target, 'a')) {
            selectPreview(e);
        }
    });

    new ImagePreviewer().attachEvents();
});

function selectPreview(ev) {
    const current = ev.target;
    const preview = closest(current, '.preview-item');
    const checkbox = preview.querySelector('input[type="checkbox"]');
    checkbox.checked = !checkbox.checked;

    ev.preventDefault();
}

function closest(el, selector) {
    while (el) {
        if (el.matches(selector)) {
            return el;
        }
        el = el.parentElement;
    }
}



class ImagePreviewer {
    constructor() {
        this.canvas = null
        this.scale = 0.25;
        this.imgX = 0;
        this.imgY = 0;
        this.positionX = 0;
        this.positionY = 0;
    }

    attachEvents() {
        this.canvas = document.getElementById('preview-canvas');
        document.body.addEventListener('mousemove', (e) => {
            if (!e.target.matches('.preview-item img')) {
                return
            }

            const img = e.target;
            this.imgX = (e.clientX - img.getBoundingClientRect().left) * (img.naturalWidth / img.width);
            this.imgY = (e.clientY - img.getBoundingClientRect().top) * (img.naturalHeight / img.height);
            this.render(img);

            this.positionX = e.pageX;
            this.positionY = e.pageY;
            this.reposition();
        });
    }

    reposition() {
        const c = this.canvas;
        const canvasWidth = c.width;
        const canvasHeight = c.height;

        let x = this.positionX;
        let y = this.positionY - canvasHeight - 10;

        if (x + c.width > window.innerWidth) {
            x = window.innerWidth - c.width;
        }

        if (y < 10) {
            y = 10;
        }

        c.style.left = `${x}px`;
        c.style.top = `${y}px`;
    }

    render(img) {
        const c = this.canvas;
        const ctx = c.getContext('2d');
        ctx.reset();
        ctx.setTransform(1, 0, 0, 1, 0, 0);
        ctx.clearRect(0, 0, c.width, c.height);

        ctx.beginPath();
        ctx.arc(c.width/2, c.height/2, c.width/2, 0, Math.PI * 2);
        ctx.clip();

        ctx.translate(c.width / 2, c.height / 2);
        ctx.scale(this.scale, this.scale);
        ctx.translate(-this.imgX, -this.imgY);
        ctx.drawImage(img, 0, 0);
    }
}