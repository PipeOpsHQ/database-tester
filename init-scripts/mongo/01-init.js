// MongoDB Initialization Script for Database Stress Testing
// This script creates collections and sample data for stress testing

// Switch to the test database
db = db.getSiblingDB('testdb');

// Drop existing collections to start fresh
db.users.drop();
db.orders.drop();
db.products.drop();
db.activity_logs.drop();
db.testcollection.drop();

print("Creating MongoDB collections and sample data for stress testing...");

// Create users collection with sample data
print("Creating users collection...");
db.users.insertMany([
    {
        _id: ObjectId(),
        username: "john_doe",
        email: "john.doe@example.com",
        first_name: "John",
        last_name: "Doe",
        age: 28,
        salary: 50000.00,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
        profile: {
            bio: "Software engineer with 5 years experience",
            skills: ["JavaScript", "Python", "Go"],
            location: "San Francisco, CA"
        },
        preferences: {
            theme: "dark",
            notifications: true,
            newsletter: false
        }
    },
    {
        _id: ObjectId(),
        username: "jane_smith",
        email: "jane.smith@example.com",
        first_name: "Jane",
        last_name: "Smith",
        age: 32,
        salary: 65000.00,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
        profile: {
            bio: "Product manager and data analyst",
            skills: ["SQL", "Analytics", "Project Management"],
            location: "New York, NY"
        },
        preferences: {
            theme: "light",
            notifications: true,
            newsletter: true
        }
    },
    {
        _id: ObjectId(),
        username: "bob_johnson",
        email: "bob.johnson@example.com",
        first_name: "Bob",
        last_name: "Johnson",
        age: 45,
        salary: 75000.00,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
        profile: {
            bio: "Senior developer and team lead",
            skills: ["Java", "Spring", "Microservices", "Docker"],
            location: "Austin, TX"
        },
        preferences: {
            theme: "dark",
            notifications: false,
            newsletter: true
        }
    },
    {
        _id: ObjectId(),
        username: "alice_brown",
        email: "alice.brown@example.com",
        first_name: "Alice",
        last_name: "Brown",
        age: 29,
        salary: 55000.00,
        is_active: true,
        created_at: new Date(),
        updated_at: new Date(),
        profile: {
            bio: "UI/UX designer with passion for user experience",
            skills: ["Figma", "Adobe Creative Suite", "User Research"],
            location: "Seattle, WA"
        },
        preferences: {
            theme: "light",
            notifications: true,
            newsletter: false
        }
    },
    {
        _id: ObjectId(),
        username: "charlie_davis",
        email: "charlie.davis@example.com",
        first_name: "Charlie",
        last_name: "Davis",
        age: 38,
        salary: 80000.00,
        is_active: false,
        created_at: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), // 30 days ago
        updated_at: new Date(),
        profile: {
            bio: "DevOps engineer specializing in cloud infrastructure",
            skills: ["AWS", "Kubernetes", "Terraform", "CI/CD"],
            location: "Denver, CO"
        },
        preferences: {
            theme: "dark",
            notifications: true,
            newsletter: true
        }
    }
]);

// Create indexes on users collection
db.users.createIndex({ "username": 1 }, { unique: true });
db.users.createIndex({ "email": 1 }, { unique: true });
db.users.createIndex({ "is_active": 1, "created_at": 1 });
db.users.createIndex({ "age": 1 });
db.users.createIndex({ "salary": 1 });
db.users.createIndex({ "profile.skills": 1 });
db.users.createIndex({ "profile.location": 1 });

print("Users collection created with " + db.users.countDocuments({}) + " documents");

// Create products collection
print("Creating products collection...");
db.products.insertMany([
    {
        _id: ObjectId(),
        sku: "LAPTOP-001",
        name: "Gaming Laptop Pro",
        description: "High-performance gaming laptop with RTX graphics",
        price: 1299.99,
        stock_quantity: 25,
        category: "Electronics",
        is_available: true,
        created_at: new Date(),
        updated_at: new Date(),
        specifications: {
            weight: 2.5,
            dimensions: { width: 35.6, height: 2.3, depth: 25.1 },
            processor: "Intel i7-12700H",
            memory: "16GB DDR4",
            storage: "1TB NVMe SSD",
            graphics: "RTX 3070"
        },
        tags: ["gaming", "laptop", "high-performance", "rtx"],
        ratings: {
            average: 4.5,
            count: 128,
            distribution: { 5: 65, 4: 45, 3: 12, 2: 4, 1: 2 }
        }
    },
    {
        _id: ObjectId(),
        sku: "PHONE-001",
        name: "Smartphone X",
        description: "Latest smartphone with advanced camera",
        price: 899.99,
        stock_quantity: 150,
        category: "Electronics",
        is_available: true,
        created_at: new Date(),
        updated_at: new Date(),
        specifications: {
            weight: 0.18,
            dimensions: { width: 7.1, height: 0.8, depth: 14.7 },
            display: "6.1-inch OLED",
            camera: "Triple 12MP system",
            battery: "3095mAh",
            storage_options: ["128GB", "256GB", "512GB"]
        },
        tags: ["smartphone", "camera", "5g", "premium"],
        ratings: {
            average: 4.2,
            count: 89,
            distribution: { 5: 38, 4: 32, 3: 15, 2: 3, 1: 1 }
        }
    },
    {
        _id: ObjectId(),
        sku: "DESK-001",
        name: "Standing Desk",
        description: "Adjustable height standing desk",
        price: 599.99,
        stock_quantity: 20,
        category: "Furniture",
        is_available: true,
        created_at: new Date(),
        updated_at: new Date(),
        specifications: {
            weight: 35.2,
            dimensions: { width: 120, height: 75, depth: 60 },
            material: "Bamboo top with steel frame",
            height_range: "73-123 cm",
            weight_capacity: "80kg"
        },
        tags: ["desk", "standing", "ergonomic", "adjustable"],
        ratings: {
            average: 4.7,
            count: 45,
            distribution: { 5: 32, 4: 10, 3: 2, 2: 1, 1: 0 }
        }
    }
]);

// Create indexes on products collection
db.products.createIndex({ "sku": 1 }, { unique: true });
db.products.createIndex({ "name": "text", "description": "text" });
db.products.createIndex({ "category": 1 });
db.products.createIndex({ "price": 1 });
db.products.createIndex({ "stock_quantity": 1 });
db.products.createIndex({ "is_available": 1 });
db.products.createIndex({ "tags": 1 });
db.products.createIndex({ "ratings.average": 1 });

print("Products collection created with " + db.products.countDocuments({}) + " documents");

// Create orders collection
print("Creating orders collection...");
const userIds = db.users.find({}, {_id: 1}).toArray().map(u => u._id);
const productData = db.products.find({}, {_id: 1, name: 1, price: 1}).toArray();

db.orders.insertMany([
    {
        _id: ObjectId(),
        user_id: userIds[0],
        order_number: "ORD-2023-001",
        total_amount: 1299.99,
        status: "delivered",
        order_date: new Date(Date.now() - 10 * 24 * 60 * 60 * 1000), // 10 days ago
        shipped_date: new Date(Date.now() - 8 * 24 * 60 * 60 * 1000),
        delivery_date: new Date(Date.now() - 5 * 24 * 60 * 60 * 1000),
        items: [
            {
                product_id: productData[0]._id,
                product_name: productData[0].name,
                quantity: 1,
                unit_price: productData[0].price,
                total_price: productData[0].price
            }
        ],
        shipping_address: {
            street: "123 Main St",
            city: "San Francisco",
            state: "CA",
            zip: "94105",
            country: "USA"
        },
        payment: {
            method: "credit_card",
            last_four: "1234",
            transaction_id: "txn_123456789"
        }
    },
    {
        _id: ObjectId(),
        user_id: userIds[1],
        order_number: "ORD-2023-002",
        total_amount: 949.98,
        status: "processing",
        order_date: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000), // 2 days ago
        items: [
            {
                product_id: productData[1]._id,
                product_name: productData[1].name,
                quantity: 1,
                unit_price: productData[1].price,
                total_price: productData[1].price
            }
        ],
        shipping_address: {
            street: "456 Oak Ave",
            city: "New York",
            state: "NY",
            zip: "10001",
            country: "USA"
        },
        payment: {
            method: "paypal",
            email: "jane.smith@example.com",
            transaction_id: "pp_987654321"
        }
    },
    {
        _id: ObjectId(),
        user_id: userIds[2],
        order_number: "ORD-2023-003",
        total_amount: 599.99,
        status: "pending",
        order_date: new Date(),
        items: [
            {
                product_id: productData[2]._id,
                product_name: productData[2].name,
                quantity: 1,
                unit_price: productData[2].price,
                total_price: productData[2].price
            }
        ],
        shipping_address: {
            street: "789 Pine Rd",
            city: "Austin",
            state: "TX",
            zip: "73301",
            country: "USA"
        },
        payment: {
            method: "bank_transfer",
            bank_name: "Austin Bank",
            transaction_id: "bt_567890123"
        }
    }
]);

// Create indexes on orders collection
db.orders.createIndex({ "order_number": 1 }, { unique: true });
db.orders.createIndex({ "user_id": 1 });
db.orders.createIndex({ "status": 1 });
db.orders.createIndex({ "order_date": 1 });
db.orders.createIndex({ "user_id": 1, "status": 1 });
db.orders.createIndex({ "total_amount": 1 });

print("Orders collection created with " + db.orders.countDocuments({}) + " documents");

// Create activity logs collection for high-volume testing
print("Creating activity_logs collection...");
const activities = [];
for (let i = 0; i < 1000; i++) {
    activities.push({
        _id: ObjectId(),
        user_id: userIds[Math.floor(Math.random() * userIds.length)],
        action: ["login", "logout", "view_product", "add_to_cart", "checkout", "search"][Math.floor(Math.random() * 6)],
        details: {
            ip_address: `192.168.1.${Math.floor(Math.random() * 255)}`,
            user_agent: "Mozilla/5.0 (compatible; stress-test)",
            session_id: `sess_${Math.random().toString(36).substr(2, 9)}`,
            metadata: {
                page: `/page_${Math.floor(Math.random() * 100)}`,
                duration: Math.floor(Math.random() * 300),
                success: Math.random() > 0.1
            }
        },
        created_at: new Date(Date.now() - Math.random() * 30 * 24 * 60 * 60 * 1000), // Random within 30 days
        score: Math.random() * 100
    });
}

// Insert activities in batches for better performance
const batchSize = 100;
for (let i = 0; i < activities.length; i += batchSize) {
    const batch = activities.slice(i, i + batchSize);
    db.activity_logs.insertMany(batch);
}

// Create indexes on activity_logs collection
db.activity_logs.createIndex({ "user_id": 1 });
db.activity_logs.createIndex({ "action": 1 });
db.activity_logs.createIndex({ "created_at": 1 });
db.activity_logs.createIndex({ "user_id": 1, "action": 1 });
db.activity_logs.createIndex({ "details.session_id": 1 });
db.activity_logs.createIndex({ "score": 1 });

print("Activity logs collection created with " + db.activity_logs.countDocuments({}) + " documents");

// Create testcollection for general stress testing
print("Creating testcollection for stress testing...");
const testDocs = [];
for (let i = 1; i <= 100; i++) {
    testDocs.push({
        _id: ObjectId(),
        test_id: i,
        name: `Test Document ${i}`,
        value: Math.random() * 1000,
        category: `category_${i % 10}`,
        active: Math.random() > 0.3,
        created_at: new Date(Date.now() - Math.random() * 365 * 24 * 60 * 60 * 1000), // Random within year
        data: {
            nested_value: Math.random() * 100,
            nested_array: [1, 2, 3, Math.floor(Math.random() * 100)],
            nested_object: {
                key1: `value_${i}`,
                key2: Math.random() * 50,
                key3: i % 2 === 0
            }
        },
        tags: [`tag_${i % 5}`, `tag_${i % 7}`, `tag_${i % 3}`]
    });
}

db.testcollection.insertMany(testDocs);

// Create indexes on testcollection
db.testcollection.createIndex({ "test_id": 1 }, { unique: true });
db.testcollection.createIndex({ "name": 1 });
db.testcollection.createIndex({ "value": 1 });
db.testcollection.createIndex({ "category": 1 });
db.testcollection.createIndex({ "active": 1 });
db.testcollection.createIndex({ "created_at": 1 });
db.testcollection.createIndex({ "tags": 1 });
db.testcollection.createIndex({ "category": 1, "active": 1 });

print("Test collection created with " + db.testcollection.countDocuments({}) + " documents");

// Create some views for complex queries
print("Creating views...");

// User order summary view
db.createView("user_order_summary", "users", [
    {
        $lookup: {
            from: "orders",
            localField: "_id",
            foreignField: "user_id",
            as: "orders"
        }
    },
    {
        $addFields: {
            total_orders: { $size: "$orders" },
            total_spent: {
                $sum: {
                    $map: {
                        input: "$orders",
                        as: "order",
                        in: "$$order.total_amount"
                    }
                }
            },
            avg_order_value: {
                $cond: {
                    if: { $gt: [{ $size: "$orders" }, 0] },
                    then: {
                        $divide: [
                            {
                                $sum: {
                                    $map: {
                                        input: "$orders",
                                        as: "order",
                                        in: "$$order.total_amount"
                                    }
                                }
                            },
                            { $size: "$orders" }
                        ]
                    },
                    else: 0
                }
            }
        }
    },
    {
        $project: {
            username: 1,
            email: 1,
            first_name: 1,
            last_name: 1,
            total_orders: 1,
            total_spent: 1,
            avg_order_value: 1,
            user_since: "$created_at"
        }
    }
]);

print("Views created successfully");

// Print summary
print("\n=== MongoDB Initialization Complete ===");
print("Database: testdb");
print("Collections created:");
print("- users: " + db.users.countDocuments({}) + " documents");
print("- products: " + db.products.countDocuments({}) + " documents");
print("- orders: " + db.orders.countDocuments({}) + " documents");
print("- activity_logs: " + db.activity_logs.countDocuments({}) + " documents");
print("- testcollection: " + db.testcollection.countDocuments({}) + " documents");
print("- user_order_summary: view created");

print("\nIndexes created on all collections for optimal query performance");
print("Sample data includes:");
print("- User profiles with nested documents");
print("- Product catalog with specifications and ratings");
print("- Order history with line items and addresses");
print("- Activity logs for high-volume testing");
print("- Test documents for general stress testing");

print("\nDatabase is ready for stress testing!");
